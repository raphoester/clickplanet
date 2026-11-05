package postgres_reset_store

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/resets"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var _ resets.Store = (*Store)(nil)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

func (s *Store) Keep(ctx context.Context, reset resets.Reset) (bool, error) {
	result, err := s.db.On(ctx).ExecContext(ctx,
		`INSERT INTO map_resets (id, at) VALUES ($1, $2) ON CONFLICT (id) DO NOTHING`, string(reset.ID), reset.At)
	if err != nil {
		return false, fmt.Errorf("failed to keep the reset %q: %w", reset.ID, err)
	}
	kept, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to keep the reset %q: %w", reset.ID, err)
	}
	return kept == 1, nil
}

func (s *Store) After(ctx context.Context, at time.Time) ([]resets.Reset, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT id, at FROM map_resets WHERE at > $1 ORDER BY at, id`, at)
	if err != nil {
		return nil, fmt.Errorf("failed to read the resets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var after []resets.Reset
	for rows.Next() {
		var (
			id   string
			kept resets.Reset
		)
		if err := rows.Scan(&id, &kept.At); err != nil {
			return nil, fmt.Errorf("failed to read a reset: %w", err)
		}
		kept.ID, kept.At = resets.ID(id), kept.At.UTC()
		after = append(after, kept)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the resets: %w", err)
	}
	return after, nil
}
