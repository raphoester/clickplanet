package cpflags

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const heardLock = 0x666c6167

// PostgresStore keeps each flag heard as one row of heard_flags, in the schema db opens.
func NewPostgresStore(db cppg.Conn) *PostgresStore {
	return &PostgresStore{db: db}
}

type PostgresStore struct {
	db cppg.Conn
}

var _ Store = (*PostgresStore)(nil)

func (s *PostgresStore) Heard(ctx context.Context, flag string) (Heard, error) {
	kept, err := keptOf(ctx, s.db.On(ctx), flag)
	if err != nil {
		return Heard{}, err
	}
	return kept.Restore()
}

func (s *PostgresStore) Change(ctx context.Context, flag string, change func(Heard) Heard) error {
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
			return err
		}
		after := change(heard).TakeSnapshot()
		if after == before {
			return nil
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO heard_flags (flag, state, decided_at) VALUES ($1, $2::heard_flag_state, $3)
			ON CONFLICT (flag) DO UPDATE SET state = EXCLUDED.state, decided_at = EXCLUDED.decided_at
		`, flag, string(after.State), after.DecidedAt.UTC()); err != nil {
			return fmt.Errorf("failed to keep the flag heard: %w", err)
		}
		return nil
	})
}

func keptOf(ctx context.Context, db cppg.Querier, flag string) (HeardSnapshot, error) {
	var state string
	var kept HeardSnapshot
	err := db.QueryRowContext(ctx, `SELECT state::text, decided_at FROM heard_flags WHERE flag = $1`, flag).
		Scan(&state, &kept.DecidedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return HeardSnapshot{}, nil
	case err != nil:
		return HeardSnapshot{}, fmt.Errorf("failed to read the flag heard: %w", err)
	}
	return HeardSnapshot{State: State(state), DecidedAt: kept.DecidedAt.UTC()}, nil
}
