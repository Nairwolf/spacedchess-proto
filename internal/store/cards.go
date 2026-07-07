package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

type CardInput struct {
	CardType   string
	FEN        string
	SideToMove string
	Details    json.RawMessage
	SourceNote string
	Tags       []string
	SetIDs     []int64
}

func (s *Store) CreateCard(ctx context.Context, userID int64, in CardInput) (*Card, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var id int64
	err = tx.QueryRow(ctx,
		`INSERT INTO cards (user_id, card_type, fen, side_to_move, details, source_note)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		userID, in.CardType, in.FEN, in.SideToMove, in.Details, in.SourceNote,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	// New cards are due immediately with default SM-2 state.
	if _, err := tx.Exec(ctx,
		`INSERT INTO review_state (card_id) VALUES ($1)`, id); err != nil {
		return nil, err
	}

	if err := replaceCardTags(ctx, tx, userID, id, in.Tags); err != nil {
		return nil, err
	}
	if err := replaceCardSets(ctx, tx, userID, id, in.SetIDs); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetCard(ctx, userID, id)
}

func (s *Store) UpdateCard(ctx context.Context, userID, id int64, in CardInput) (*Card, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	ct, err := tx.Exec(ctx,
		`UPDATE cards SET card_type = $1, fen = $2, side_to_move = $3, details = $4,
		        source_note = $5, updated_at = now()
		 WHERE id = $6 AND user_id = $7`,
		in.CardType, in.FEN, in.SideToMove, in.Details, in.SourceNote, id, userID)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, ErrNotFound
	}

	if err := replaceCardTags(ctx, tx, userID, id, in.Tags); err != nil {
		return nil, err
	}
	if err := replaceCardSets(ctx, tx, userID, id, in.SetIDs); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetCard(ctx, userID, id)
}

func (s *Store) DeleteCard(ctx context.Context, userID, id int64) error {
	ct, err := s.pool.Exec(ctx,
		`DELETE FROM cards WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

const cardSelect = `
	SELECT c.id, c.card_type, c.fen, c.side_to_move, c.details, c.source_note,
	       c.created_at, c.updated_at,
	       r.easiness_factor, r.interval_days, r.repetitions, r.due_at, r.last_reviewed_at
	FROM cards c
	JOIN review_state r ON r.card_id = c.id`

func scanCard(row pgx.Row) (*Card, error) {
	c := &Card{Tags: []string{}, SetIDs: []int64{}}
	err := row.Scan(&c.ID, &c.CardType, &c.FEN, &c.SideToMove, &c.Details, &c.SourceNote,
		&c.CreatedAt, &c.UpdatedAt,
		&c.Review.Easiness, &c.Review.IntervalDays, &c.Review.Repetitions,
		&c.Review.DueAt, &c.Review.LastReviewedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Store) GetCard(ctx context.Context, userID, id int64) (*Card, error) {
	c, err := scanCard(s.pool.QueryRow(ctx,
		cardSelect+` WHERE c.id = $1 AND c.user_id = $2`, id, userID))
	if err != nil {
		return nil, err
	}
	if err := s.loadCardMeta(ctx, []*Card{c}); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Store) ListCards(ctx context.Context, userID int64, f CardFilter) ([]*Card, error) {
	where := []string{"c.user_id = $1"}
	args := []any{userID}
	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if f.CardType != "" {
		where = append(where, "c.card_type = "+arg(f.CardType))
	}
	if f.Tag != "" {
		where = append(where, `EXISTS (
			SELECT 1 FROM card_tags ct JOIN tags t ON t.id = ct.tag_id
			WHERE ct.card_id = c.id AND lower(t.name) = lower(`+arg(f.Tag)+`))`)
	}
	if f.SetID != 0 {
		where = append(where, `EXISTS (
			SELECT 1 FROM set_cards sc WHERE sc.card_id = c.id AND sc.set_id = `+arg(f.SetID)+`)`)
	}
	if f.Search != "" {
		p := arg("%" + f.Search + "%")
		where = append(where, `(c.source_note ILIKE `+p+` OR c.details::text ILIKE `+p+`)`)
	}

	order := " ORDER BY c.created_at DESC, c.id DESC"
	if f.DueOnly {
		where = append(where, "r.due_at <= now()")
		order = " ORDER BY r.due_at ASC, c.id ASC"
	}

	rows, err := s.pool.Query(ctx,
		cardSelect+" WHERE "+strings.Join(where, " AND ")+order, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cards := []*Card{}
	for rows.Next() {
		c, err := scanCard(rows)
		if err != nil {
			return nil, err
		}
		cards = append(cards, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := s.loadCardMeta(ctx, cards); err != nil {
		return nil, err
	}
	return cards, nil
}

// loadCardMeta batch-loads tags and set memberships for the given cards.
func (s *Store) loadCardMeta(ctx context.Context, cards []*Card) error {
	if len(cards) == 0 {
		return nil
	}
	byID := make(map[int64]*Card, len(cards))
	ids := make([]int64, 0, len(cards))
	for _, c := range cards {
		byID[c.ID] = c
		ids = append(ids, c.ID)
	}

	rows, err := s.pool.Query(ctx,
		`SELECT ct.card_id, t.name FROM card_tags ct
		 JOIN tags t ON t.id = ct.tag_id
		 WHERE ct.card_id = ANY($1) ORDER BY t.name`, ids)
	if err != nil {
		return err
	}
	for rows.Next() {
		var cardID int64
		var name string
		if err := rows.Scan(&cardID, &name); err != nil {
			rows.Close()
			return err
		}
		byID[cardID].Tags = append(byID[cardID].Tags, name)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	rows, err = s.pool.Query(ctx,
		`SELECT card_id, set_id FROM set_cards WHERE card_id = ANY($1) ORDER BY set_id`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cardID, setID int64
		if err := rows.Scan(&cardID, &setID); err != nil {
			return err
		}
		byID[cardID].SetIDs = append(byID[cardID].SetIDs, setID)
	}
	return rows.Err()
}

// replaceCardTags sets the card's tags to exactly names, creating tags that
// don't exist yet. Names are trimmed and deduplicated case-insensitively.
func replaceCardTags(ctx context.Context, tx pgx.Tx, userID, cardID int64, names []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM card_tags WHERE card_id = $1`, cardID); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, name := range names {
		name = strings.Join(strings.Fields(name), " ")
		if name == "" || seen[strings.ToLower(name)] {
			continue
		}
		seen[strings.ToLower(name)] = true
		var tagID int64
		err := tx.QueryRow(ctx,
			`INSERT INTO tags (user_id, name) VALUES ($1, $2)
			 ON CONFLICT (user_id, lower(name)) DO UPDATE SET name = tags.name
			 RETURNING id`,
			userID, name,
		).Scan(&tagID)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO card_tags (card_id, tag_id) VALUES ($1, $2)`, cardID, tagID); err != nil {
			return err
		}
	}
	return nil
}

// replaceCardSets sets the card's set memberships. Set IDs not owned by the
// user are silently ignored (they can't be referenced cross-user).
func replaceCardSets(ctx context.Context, tx pgx.Tx, userID, cardID int64, setIDs []int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM set_cards WHERE card_id = $1`, cardID); err != nil {
		return err
	}
	if len(setIDs) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx,
		`INSERT INTO set_cards (set_id, card_id)
		 SELECT s.id, $1 FROM sets s WHERE s.id = ANY($2) AND s.user_id = $3
		 ON CONFLICT DO NOTHING`,
		cardID, setIDs, userID)
	return err
}
