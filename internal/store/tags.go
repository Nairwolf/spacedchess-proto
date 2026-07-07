package store

import (
	"context"
	"strings"
)

func (s *Store) ListTags(ctx context.Context, userID int64) ([]Tag, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT t.id, t.name, count(ct.card_id)
		 FROM tags t
		 LEFT JOIN card_tags ct ON ct.tag_id = t.id
		 WHERE t.user_id = $1
		 GROUP BY t.id, t.name
		 ORDER BY lower(t.name)`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := []Tag{}
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.CardCount); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (s *Store) CreateTag(ctx context.Context, userID int64, name string) (*Tag, error) {
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return nil, ErrNotFound
	}
	t := &Tag{Name: name}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO tags (user_id, name) VALUES ($1, $2) RETURNING id`,
		userID, name,
	).Scan(&t.ID)
	if isUniqueViolation(err) {
		return nil, ErrConflict
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Store) RenameTag(ctx context.Context, userID, id int64, name string) error {
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return ErrConflict
	}
	ct, err := s.pool.Exec(ctx,
		`UPDATE tags SET name = $1 WHERE id = $2 AND user_id = $3`, name, id, userID)
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

func (s *Store) DeleteTag(ctx context.Context, userID, id int64) error {
	ct, err := s.pool.Exec(ctx,
		`DELETE FROM tags WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
