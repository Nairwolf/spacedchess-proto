// Package store is the PostgreSQL data-access layer. Every query is scoped
// by user_id (ARCHITECTURE.md §7): no data is ever readable across users.
package store

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("already exists")
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type ReviewState struct {
	Easiness       float64    `json:"easiness_factor"`
	IntervalDays   int        `json:"interval_days"`
	Repetitions    int        `json:"repetitions"`
	DueAt          time.Time  `json:"due_at"`
	LastReviewedAt *time.Time `json:"last_reviewed_at"`
}

type Card struct {
	ID         int64           `json:"id"`
	CardType   string          `json:"card_type"`
	FEN        string          `json:"fen"`
	SideToMove string          `json:"side_to_move"`
	Details    json.RawMessage `json:"details"`
	SourceNote string          `json:"source_note"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	Tags       []string        `json:"tags"`
	SetIDs     []int64         `json:"set_ids"`
	Review     ReviewState     `json:"review"`
}

type Tag struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	CardCount int    `json:"card_count"`
}

type Set struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	CardCount int       `json:"card_count"`
}

// CardFilter narrows card listings; zero values mean "no filter".
type CardFilter struct {
	CardType string
	Tag      string // tag name, case-insensitive
	SetID    int64
	Search   string // substring match on source_note and details text
	DueOnly  bool   // only cards with review_state.due_at <= now
}
