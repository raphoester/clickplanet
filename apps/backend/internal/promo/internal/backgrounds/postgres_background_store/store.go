package postgres_background_store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const foreignKeyViolation = "23503"

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

// A background kept again takes its new name.
func (s *Store) Keep(ctx context.Context, background backgrounds.Background) error {
	kept := background.TakeSnapshot()
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO backgrounds (background_id, name, media_type, size_bytes, kept_at) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (background_id) DO UPDATE SET name = excluded.name
	`, uuid.UUID(kept.File.ID), kept.Name, string(kept.File.MediaType), kept.File.Size, kept.KeptAt); err != nil {
		return fmt.Errorf("failed to keep the background %s: %w", kept.File.ID, err)
	}
	return nil
}

// A background a card shows stays, and the refusal names the sets: the cards' foreign key holds it too.
func (s *Store) Remove(ctx context.Context, id backgrounds.BackgroundID) error {
	sets, err := s.showing(ctx, id)
	if err != nil {
		return err
	}
	if len(sets) > 0 {
		return shown(id, sets)
	}

	res, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM backgrounds WHERE background_id = $1`, uuid.UUID(id))
	var pqErr *pq.Error
	switch {
	case errors.As(err, &pqErr) && pqErr.Code == foreignKeyViolation:
		return shown(id, nil)
	case err != nil:
		return fmt.Errorf("failed to remove the background %s: %w", id, err)
	}
	removed, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read whether the background %s was removed: %w", id, err)
	}
	if removed == 0 {
		return fmt.Errorf("%w: %s", backgrounds.ErrNoBackground, id)
	}
	return nil
}

func (s *Store) showing(ctx context.Context, id backgrounds.BackgroundID) ([]string, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx,
		`SELECT DISTINCT rule_set FROM cards WHERE background_id = $1 ORDER BY rule_set`, uuid.UUID(id))
	if err != nil {
		return nil, fmt.Errorf("failed to read the sets showing the background %s: %w", id, err)
	}
	defer func() { _ = rows.Close() }()
	var sets []string
	for rows.Next() {
		var set string
		if err := rows.Scan(&set); err != nil {
			return nil, fmt.Errorf("failed to read a set showing the background %s: %w", id, err)
		}
		sets = append(sets, set)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the sets showing the background %s: %w", id, err)
	}
	return sets, nil
}

func shown(id backgrounds.BackgroundID, sets []string) error {
	if len(sets) == 0 {
		return fmt.Errorf("%w: %s", backgrounds.ErrShown, id)
	}
	return fmt.Errorf("%w: %s, on the cards of %s", backgrounds.ErrShown, id, strings.Join(sets, ", "))
}
