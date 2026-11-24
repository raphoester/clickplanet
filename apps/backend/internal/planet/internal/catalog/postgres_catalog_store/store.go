package postgres_catalog_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const heardLock = 0x666c6167

// Store keeps each flag heard as one row of planet.heard_flags.
func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ catalog.Store = (*Store)(nil)

func (s *Store) Heard(ctx context.Context, flag string) (catalog.Heard, error) {
	kept, err := keptOf(ctx, s.db.On(ctx), flag)
	if err != nil {
		return catalog.Heard{}, err
	}
	heard, err := kept.Restore()
	if err != nil {
		return catalog.Heard{}, fmt.Errorf("failed to restore the flag heard: %w", err)
	}
	return heard, nil
}

func (s *Store) Change(ctx context.Context, flag string, change func(catalog.Heard) catalog.Heard) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1, hashtext($2))`, heardLock, flag); err != nil {
			return fmt.Errorf("failed to lock the flag heard: %w", err)
		}
		before, err := keptOf(ctx, tx, flag)
		if err != nil {
			return err
		}
		heard, err := before.Restore()
		if err != nil {
			return fmt.Errorf("failed to restore the flag heard: %w", err)
		}
		after := change(heard).TakeSnapshot()
		if after == before {
			return nil
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO heard_flags (flag, state, decided_at, kept) VALUES ($1, $2::heard_flag_state, $3, $4)
			ON CONFLICT (flag) DO UPDATE SET state = EXCLUDED.state, decided_at = EXCLUDED.decided_at, kept = EXCLUDED.kept
		`, flag, string(after.State), after.DecidedAt.UTC(), after.Kept); err != nil {
			return fmt.Errorf("failed to keep the flag heard: %w", err)
		}
		return nil
	})
}

func keptOf(ctx context.Context, db cppg.Querier, flag string) (catalog.HeardSnapshot, error) {
	var state string
	var kept catalog.HeardSnapshot
	err := db.QueryRowContext(ctx, `SELECT state::text, decided_at, kept FROM heard_flags WHERE flag = $1`, flag).
		Scan(&state, &kept.DecidedAt, &kept.Kept)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return catalog.HeardSnapshot{}, nil
	case err != nil:
		return catalog.HeardSnapshot{}, fmt.Errorf("failed to read the flag heard: %w", err)
	}
	return catalog.HeardSnapshot{State: catalog.State(state), DecidedAt: kept.DecidedAt.UTC(), Kept: kept.Kept}, nil
}
