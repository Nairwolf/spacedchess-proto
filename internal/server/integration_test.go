//go:build integration

// End-to-end test of the core product loop against a real (embedded)
// PostgreSQL: register → create cards of all three types → browse/filter →
// due queue → review → SM-2 rescheduling → user isolation.
//
// Run with: go test -tags=integration ./internal/server/
// (downloads PostgreSQL binaries on first run; needs network)
package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nairwolf/spacedchess/internal/migrate"
	"github.com/nairwolf/spacedchess/internal/store"
)

const (
	pgPort     = 54987
	startFEN   = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	scholarFEN = "r1bqkb1r/pppp1ppp/2n2n2/4p2Q/2B1P3/8/PPPP1PPP/RNB1K1NR w KQkq - 4 4"
)

type client struct {
	t    *testing.T
	http *http.Client
	base string
}

func newClient(t *testing.T, base string) *client {
	jar, _ := cookiejar.New(nil)
	return &client{t: t, http: &http.Client{Jar: jar}, base: base}
}

// do sends a JSON request, asserts the status code, and decodes into out.
func (c *client) do(method, path string, body any, wantStatus int, out any) {
	c.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req, err := http.NewRequest(method, c.base+path, &buf)
	if err != nil {
		c.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	var raw json.RawMessage
	json.NewDecoder(resp.Body).Decode(&raw)
	if resp.StatusCode != wantStatus {
		c.t.Fatalf("%s %s: status %d, want %d (body: %s)", method, path, resp.StatusCode, wantStatus, raw)
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			c.t.Fatalf("%s %s: decode: %v (body: %s)", method, path, err, raw)
		}
	}
}

func TestCoreLoop(t *testing.T) {
	runtimeDir := filepath.Join(os.TempDir(), "spacedchess-epg")
	epg := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Port(pgPort).
		RuntimePath(runtimeDir).
		Database("spacedchess"))
	if err := epg.Start(); err != nil {
		t.Fatalf("start embedded postgres: %v", err)
	}
	defer epg.Stop()

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, fmt.Sprintf(
		"postgres://postgres:postgres@localhost:%d/spacedchess?sslmode=disable", pgPort))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := migrate.Run(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	srv := New(store.New(pool), Options{AllowRegistration: true},
		slog.New(slog.NewTextHandler(os.Stderr, nil)))
	ts := httptest.NewServer(srv.Routes())
	defer ts.Close()

	c := newClient(t, ts.URL)

	// --- Auth ---
	c.do("POST", "/api/auth/register", map[string]string{"username": "magnus", "password": "short"},
		http.StatusBadRequest, nil)
	var user store.User
	c.do("POST", "/api/auth/register", map[string]string{"username": "magnus", "password": "hunter2hunter2"},
		http.StatusCreated, &user)
	if user.Username != "magnus" {
		t.Fatalf("unexpected user: %+v", user)
	}
	c.do("POST", "/api/auth/register", map[string]string{"username": "Magnus", "password": "hunter2hunter2"},
		http.StatusConflict, nil) // case-insensitive uniqueness
	c.do("GET", "/api/auth/me", nil, http.StatusOK, &user)

	anon := newClient(t, ts.URL)
	anon.do("GET", "/api/cards", nil, http.StatusUnauthorized, nil)
	anon.do("POST", "/api/auth/login", map[string]string{"username": "magnus", "password": "wrongwrong"},
		http.StatusUnauthorized, nil)

	// --- Sets & tags ---
	var set store.Set
	c.do("POST", "/api/sets", map[string]string{"name": "Najdorf mistakes"}, http.StatusCreated, &set)
	var tag store.Tag
	c.do("POST", "/api/tags", map[string]string{"name": "endgame"}, http.StatusCreated, &tag)

	// --- Card creation, all three types ---
	var tactical store.Card
	c.do("POST", "/api/cards", map[string]any{
		"card_type": "tactical_opportunity",
		"fen":       scholarFEN,
		"details":   map[string]any{"solution": []string{"Qxf7#"}},
		"tags":      []string{"mate", "Back-Rank"},
		"set_ids":   []int64{set.ID},
	}, http.StatusCreated, &tactical)
	if tactical.SideToMove != "w" || len(tactical.Tags) != 2 || len(tactical.SetIDs) != 1 {
		t.Fatalf("tactical card: %+v", tactical)
	}
	if tactical.Review.Easiness != 2.5 || tactical.Review.Repetitions != 0 {
		t.Fatalf("new card review state: %+v", tactical.Review)
	}

	// Illegal solution is rejected by server-side chess validation.
	c.do("POST", "/api/cards", map[string]any{
		"card_type": "tactical_opportunity",
		"fen":       scholarFEN,
		"details":   map[string]any{"solution": []string{"Qxa8"}},
	}, http.StatusBadRequest, nil)

	var blunder store.Card
	c.do("POST", "/api/cards", map[string]any{
		"card_type": "blunder",
		"fen":       startFEN,
		"details": map[string]any{
			"intended_move":       "f3",
			"refutation":          []string{"e5"},
			"correct_alternative": []string{"e4"},
		},
		"source_note": "vs. someone, 2026-06-01",
	}, http.StatusCreated, &blunder)

	var strategic store.Card
	c.do("POST", "/api/cards", map[string]any{
		"card_type": "strategic_mistake",
		"fen":       startFEN,
		"details": map[string]any{
			"question": "What plan is wrong here?",
			"answer":   "The plan itself.",
		},
		"tags": []string{"endgame"},
	}, http.StatusCreated, &strategic)

	// --- Browse & filter ---
	var cards []store.Card
	c.do("GET", "/api/cards", nil, http.StatusOK, &cards)
	if len(cards) != 3 {
		t.Fatalf("want 3 cards, got %d", len(cards))
	}
	c.do("GET", "/api/cards?type=blunder", nil, http.StatusOK, &cards)
	if len(cards) != 1 || cards[0].ID != blunder.ID {
		t.Fatalf("type filter: %+v", cards)
	}
	c.do("GET", "/api/cards?tag=ENDGAME", nil, http.StatusOK, &cards)
	if len(cards) != 1 || cards[0].ID != strategic.ID {
		t.Fatalf("tag filter: %+v", cards)
	}
	c.do("GET", fmt.Sprintf("/api/cards?set_id=%d", set.ID), nil, http.StatusOK, &cards)
	if len(cards) != 1 || cards[0].ID != tactical.ID {
		t.Fatalf("set filter: %+v", cards)
	}

	// --- Due queue & review ---
	c.do("GET", "/api/review/due", nil, http.StatusOK, &cards)
	if len(cards) != 3 {
		t.Fatalf("want 3 due, got %d", len(cards))
	}
	c.do("GET", fmt.Sprintf("/api/review/due?set_id=%d", set.ID), nil, http.StatusOK, &cards)
	if len(cards) != 1 {
		t.Fatalf("want 1 due in set, got %d", len(cards))
	}

	var state store.ReviewState
	c.do("POST", fmt.Sprintf("/api/cards/%d/review", tactical.ID),
		map[string]bool{"correct": true}, http.StatusOK, &state)
	if state.IntervalDays != 1 || state.Repetitions != 1 || state.Easiness != 2.6 {
		t.Fatalf("after correct review: %+v", state)
	}
	if !state.DueAt.After(time.Now().Add(23 * time.Hour)) {
		t.Fatalf("due_at should be ~1 day out: %v", state.DueAt)
	}

	c.do("POST", fmt.Sprintf("/api/cards/%d/review", blunder.ID),
		map[string]bool{"correct": false}, http.StatusOK, &state)
	if state.IntervalDays != 1 || state.Repetitions != 0 || math.Abs(state.Easiness-2.18) > 1e-9 {
		t.Fatalf("after incorrect review: %+v", state)
	}

	// Reviewed cards leave the due queue.
	c.do("GET", "/api/review/due", nil, http.StatusOK, &cards)
	if len(cards) != 1 || cards[0].ID != strategic.ID {
		t.Fatalf("due after reviews: %+v", cards)
	}

	// --- Card update ---
	c.do("PUT", fmt.Sprintf("/api/cards/%d", strategic.ID), map[string]any{
		"card_type": "strategic_mistake",
		"fen":       startFEN,
		"details":   map[string]any{"question": "Updated?", "answer": "Yes."},
		"tags":      []string{"opening"},
	}, http.StatusOK, &strategic)
	if len(strategic.Tags) != 1 || strategic.Tags[0] != "opening" {
		t.Fatalf("updated tags: %+v", strategic.Tags)
	}

	// --- Set membership endpoints ---
	c.do("PUT", fmt.Sprintf("/api/sets/%d/cards/%d", set.ID, blunder.ID), nil, http.StatusOK, nil)
	c.do("GET", fmt.Sprintf("/api/cards?set_id=%d", set.ID), nil, http.StatusOK, &cards)
	if len(cards) != 2 {
		t.Fatalf("set should have 2 cards, got %d", len(cards))
	}
	c.do("DELETE", fmt.Sprintf("/api/sets/%d/cards/%d", set.ID, blunder.ID), nil, http.StatusOK, nil)
	c.do("PUT", fmt.Sprintf("/api/sets/%d/cards/999999", set.ID), nil, http.StatusNotFound, nil)

	// --- Tag management ---
	var tags []store.Tag
	c.do("GET", "/api/tags", nil, http.StatusOK, &tags)
	// endgame (now unused), mate, Back-Rank, opening
	if len(tags) != 4 {
		t.Fatalf("want 4 tags, got %+v", tags)
	}
	c.do("PATCH", fmt.Sprintf("/api/tags/%d", tag.ID), map[string]string{"name": "endings"},
		http.StatusOK, nil)
	c.do("DELETE", fmt.Sprintf("/api/tags/%d", tag.ID), nil, http.StatusOK, nil)

	// --- User isolation ---
	c2 := newClient(t, ts.URL)
	c2.do("POST", "/api/auth/register", map[string]string{"username": "hikaru", "password": "hunter2hunter2"},
		http.StatusCreated, nil)
	c2.do("GET", "/api/cards", nil, http.StatusOK, &cards)
	if len(cards) != 0 {
		t.Fatalf("user2 should see no cards, got %d", len(cards))
	}
	c2.do("GET", fmt.Sprintf("/api/cards/%d", tactical.ID), nil, http.StatusNotFound, nil)
	c2.do("POST", fmt.Sprintf("/api/cards/%d/review", tactical.ID),
		map[string]bool{"correct": true}, http.StatusNotFound, nil)
	c2.do("DELETE", fmt.Sprintf("/api/cards/%d", tactical.ID), nil, http.StatusNotFound, nil)

	// --- Logout ---
	c.do("POST", "/api/auth/logout", nil, http.StatusOK, nil)
	c.do("GET", "/api/auth/me", nil, http.StatusUnauthorized, nil)

	// --- Delete card ---
	c2login := newClient(t, ts.URL)
	c2login.do("POST", "/api/auth/login", map[string]string{"username": "magnus", "password": "hunter2hunter2"},
		http.StatusOK, nil)
	c2login.do("DELETE", fmt.Sprintf("/api/cards/%d", tactical.ID), nil, http.StatusOK, nil)
	c2login.do("GET", "/api/cards", nil, http.StatusOK, &cards)
	if len(cards) != 2 {
		t.Fatalf("after delete want 2 cards, got %d", len(cards))
	}
}
