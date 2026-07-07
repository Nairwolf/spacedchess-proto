// Package server exposes the REST JSON API (ARCHITECTURE.md §4) and serves
// the built frontend as a static SPA.
package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/nairwolf/spacedchess/internal/store"
)

type Server struct {
	store             *store.Store
	staticDir         string
	secureCookies     bool
	allowRegistration bool
	log               *slog.Logger
}

type Options struct {
	StaticDir         string
	SecureCookies     bool
	AllowRegistration bool
}

func New(st *store.Store, opts Options, log *slog.Logger) *Server {
	return &Server{
		store:             st,
		staticDir:         opts.StaticDir,
		secureCookies:     opts.SecureCookies,
		allowRegistration: opts.AllowRegistration,
		log:               log,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", s.handleLogout)
	mux.Handle("GET /api/auth/me", s.auth(s.handleMe))

	mux.Handle("POST /api/cards", s.auth(s.handleCreateCard))
	mux.Handle("GET /api/cards", s.auth(s.handleListCards))
	mux.Handle("GET /api/cards/{id}", s.auth(s.handleGetCard))
	mux.Handle("PUT /api/cards/{id}", s.auth(s.handleUpdateCard))
	mux.Handle("DELETE /api/cards/{id}", s.auth(s.handleDeleteCard))

	mux.Handle("GET /api/tags", s.auth(s.handleListTags))
	mux.Handle("POST /api/tags", s.auth(s.handleCreateTag))
	mux.Handle("PATCH /api/tags/{id}", s.auth(s.handleRenameTag))
	mux.Handle("DELETE /api/tags/{id}", s.auth(s.handleDeleteTag))

	mux.Handle("GET /api/sets", s.auth(s.handleListSets))
	mux.Handle("POST /api/sets", s.auth(s.handleCreateSet))
	mux.Handle("PATCH /api/sets/{id}", s.auth(s.handleRenameSet))
	mux.Handle("DELETE /api/sets/{id}", s.auth(s.handleDeleteSet))
	mux.Handle("PUT /api/sets/{id}/cards/{cardId}", s.auth(s.handleAddCardToSet))
	mux.Handle("DELETE /api/sets/{id}/cards/{cardId}", s.auth(s.handleRemoveCardFromSet))

	mux.Handle("GET /api/review/due", s.auth(s.handleDueCards))
	mux.Handle("POST /api/cards/{id}/review", s.auth(s.handleSubmitReview))

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		s.jsonError(w, http.StatusNotFound, "not found")
	})

	mux.Handle("/", s.spaHandler())

	return mux
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (s *Server) jsonError(w http.ResponseWriter, status int, msg string) {
	s.writeJSON(w, status, map[string]string{"error": msg})
}

// storeError maps store-layer sentinel errors to HTTP responses.
func (s *Server) storeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		s.jsonError(w, http.StatusNotFound, "not found")
	case errors.Is(err, store.ErrConflict):
		s.jsonError(w, http.StatusConflict, "already exists")
	default:
		s.log.Error("internal error", "err", err)
		s.jsonError(w, http.StatusInternalServerError, "internal error")
	}
}

func (s *Server) decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(v); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

func pathID(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}
