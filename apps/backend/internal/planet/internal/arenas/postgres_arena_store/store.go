package postgres_arena_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var _ arenas.Store = (*Store)(nil)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

func (s *Store) Find(ctx context.Context, id arenas.ID) (arenas.Arena, error) {
	arena := arenas.Arena{ID: id}
	var (
		endsAt, openedAt sql.NullTime
		closed           bool
	)
	err := s.db.On(ctx).QueryRowContext(ctx, `
		SELECT a.ends_at, a.opened_at, EXISTS (SELECT 1 FROM closed_arenas WHERE id = $1)
		FROM (SELECT 1) AS one
		LEFT JOIN arenas a ON a.id = $1
	`, string(id)).Scan(&endsAt, &openedAt, &closed)
	if err != nil {
		return arenas.Arena{}, fmt.Errorf("failed to read arena %s: %w", id, err)
	}
	if openedAt.Valid {
		arena.Opened = true
		arena.Opening = arenas.Opening{ID: id, EndsAt: endsAt.Time.UTC(), OpenedAt: openedAt.Time.UTC()}
	}
	arena.Closed = closed
	return arena, nil
}

func (s *Store) Playing(ctx context.Context) (arenas.Arena, error) {
	var (
		id               string
		endsAt, openedAt time.Time
	)
	err := s.db.On(ctx).QueryRowContext(ctx, `
		SELECT id, ends_at, opened_at FROM arenas a
		WHERE NOT EXISTS (SELECT 1 FROM closed_arenas c WHERE c.id = a.id)
		ORDER BY opened_at DESC
		LIMIT 1
	`).Scan(&id, &endsAt, &openedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return arenas.Arena{}, nil
	}
	if err != nil {
		return arenas.Arena{}, fmt.Errorf("failed to read the arena open: %w", err)
	}
	opening := arenas.Opening{ID: arenas.ID(id), EndsAt: endsAt.UTC(), OpenedAt: openedAt.UTC()}
	return arenas.Arena{ID: opening.ID, Opening: opening, Opened: true}, nil
}

func (s *Store) Open(ctx context.Context, opening arenas.Opening) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `INSERT INTO arenas (id, ends_at, opened_at) VALUES ($1, $2, $3)`,
		string(opening.ID), opening.EndsAt, opening.OpenedAt); err != nil {
		return fmt.Errorf("failed to keep arena %s open: %w", opening.ID, err)
	}
	return nil
}

func (s *Store) Close(ctx context.Context, closing arenas.Closing) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `INSERT INTO closed_arenas (id, closed_at) VALUES ($1, $2)`,
		string(closing.ID), closing.ClosedAt); err != nil {
		return fmt.Errorf("failed to keep arena %s closed: %w", closing.ID, err)
	}
	return nil
}
