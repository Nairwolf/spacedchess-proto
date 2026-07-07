package store

import (
	"context"
	"strings"
)

func (s *Store) ListSets(ctx context.Context, userID int64) ([]Set, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT s.id, s.name, s.created_at, count(sc.card_id)
		 FROM sets s
		 LEFT JOIN set_cards sc ON sc.set_id = s.id
		 WHERE s.user_id = $1
		 GROUP BY s.id, s.name, s.created_at
		 ORDER BY lower(s.name)`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sets := []Set{}
	for rows.Next() {
		var set Set
		if err := rows.Scan(&set.ID, &set.Name, &set.CreatedAt, &set.CardCount); err != nil {
			return nil, err
		}
		sets = append(sets, set)
	}
	return sets, rows.Err()
}

func (s *Store) CreateSet(ctx context.Context, userID int64, name string) (*Set, error) {
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return nil, ErrConflict
	}
	set := &Set{Name: name}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO sets (user_id, name) VALUES ($1, $2) RETURNING id, created_at`,
		userID, name,
	).Scan(&set.ID, &set.CreatedAt)
	if isUniqueViolation(err) {
		return nil, ErrConflict
	}
	if err != nil {
		return nil, err
	}
	return set, nil
}

func (s *Store) RenameSet(ctx context.Context, userID, id int64, name string) error {
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return ErrConflict
	}
	ct, err := s.pool.Exec(ctx,
		`UPDATE sets SET name = $1 WHERE id = $2 AND user_id = $3`, name, id, userID)
	if isUniqueViolation(err) {
		return ErrConflict
	}
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteSet(ctx context.Context, userID, id int64) error {
	ct, err := s.pool.Exec(ctx,
		`DELETE FROM sets WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// AddCardToSet verifies both the set and the card belong to the user.
func (s *Store) AddCardToSet(ctx context.Context, userID, setID, cardID int64) error {
	ct, err := s.pool.Exec(ctx,
		`INSERT INTO set_cards (set_id, card_id)
		 SELECT s.id, c.id FROM sets s, cards c
		 WHERE s.id = $1 AND s.user_id = $3 AND c.id = $2 AND c.user_id = $3
		 ON CONFLICT DO NOTHING`,
		setID, cardID, userID)
	if err != nil {
		return err
	}
	// 0 rows can mean "already in set" (fine) or "not found"; distinguish.
	if ct.RowsAffected() == 0 {
		var exists bool
		err := s.pool.QueryRow(ctx,
			`SELECT EXISTS (
				SELECT 1 FROM sets s, cards c
				WHERE s.id = $1 AND s.user_id = $3 AND c.id = $2 AND c.user_id = $3)`,
			setID, cardID, userID,
		).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
	}
	return nil
}

func (s *Store) RemoveCardFromSet(ctx context.Context, userID, setID, cardID int64) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM set_cards sc USING sets s
		 WHERE sc.set_id = s.id AND s.user_id = $3 AND sc.set_id = $1 AND sc.card_id = $2`,
		setID, cardID, userID)
	return err
}
