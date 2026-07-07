package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nairwolf/spacedchess/internal/srs"
)

// SubmitReview applies one SM-2 review to a card (which must belong to the
// user), updates review_state, and appends to review_log. Returns the new
// scheduling state.
func (s *Store) SubmitReview(ctx context.Context, userID, cardID int64, correct bool, now time.Time) (*ReviewState, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var cur srs.State
	err = tx.QueryRow(ctx,
		`SELECT r.easiness_factor, r.interval_days, r.repetitions, r.due_at
		 FROM review_state r JOIN cards c ON c.id = r.card_id
		 WHERE r.card_id = $1 AND c.user_id = $2
		 FOR UPDATE OF r`,
		cardID, userID,
	).Scan(&cur.Easiness, &cur.IntervalDays, &cur.Repetitions, &cur.DueAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	next := srs.Review(cur, correct, now)

	if _, err := tx.Exec(ctx,
		`UPDATE review_state
		 SET easiness_factor = $1, interval_days = $2, repetitions = $3,
		     due_at = $4, last_reviewed_at = $5
		 WHERE card_id = $6`,
		next.Easiness, next.IntervalDays, next.Repetitions, next.DueAt, now, cardID); err != nil {
		return nil, err
	}

	grade := 0
	if correct {
		grade = 1
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO review_log (card_id, reviewed_at, grade, resulting_interval_days)
		 VALUES ($1, $2, $3, $4)`,
		cardID, now, grade, next.IntervalDays); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &ReviewState{
		Easiness:       next.Easiness,
		IntervalDays:   next.IntervalDays,
		Repetitions:    next.Repetitions,
		DueAt:          next.DueAt,
		LastReviewedAt: &now,
	}, nil
}

// DueCount returns how many of the user's cards are due now.
func (s *Store) DueCount(ctx context.Context, userID int64) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM cards c
		 JOIN review_state r ON r.card_id = c.id
		 WHERE c.user_id = $1 AND r.due_at <= now()`, userID).Scan(&n)
	return n, err
}
