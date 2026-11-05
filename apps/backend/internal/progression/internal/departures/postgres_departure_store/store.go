package postgres_departure_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/departures"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const departureLock = 0x64657061

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ departures.Store = (*Store)(nil)

func (s *Store) Departure(ctx context.Context, account accounts.AccountID) (departures.Departure, error) {
	kept, err := keptOf(ctx, s.db.On(ctx), account)
	if err != nil {
		return departures.Departure{}, err
	}
	return kept.Restore(), nil
}

func (s *Store) Change(ctx context.Context, account accounts.AccountID, change func(departures.Departure) departures.Departure) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1, hashtext($2))`, departureLock, account.String()); err != nil {
			return fmt.Errorf("failed to lock the departure: %w", err)
		}

		before, err := keptOf(ctx, tx, account)
		if err != nil {
			return err
		}
		after := change(before.Restore()).TakeSnapshot()
		if after == before {
			return nil
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO departures (account_id, leaving, decided_at) VALUES ($1, $2, $3)
			ON CONFLICT (account_id) DO UPDATE SET leaving = EXCLUDED.leaving, decided_at = EXCLUDED.decided_at
		`, uuid.UUID(account), after.Leaving, after.DecidedAt.UTC()); err != nil {
			return fmt.Errorf("failed to write the departure: %w", err)
		}
		return nil
	})
}

func (s *Store) Forget(ctx context.Context, account accounts.AccountID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM departures WHERE account_id = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to forget the departure: %w", err)
	}
	return nil
}

func keptOf(ctx context.Context, db cppg.Querier, account accounts.AccountID) (departures.Snapshot, error) {
	var kept departures.Snapshot
	err := db.QueryRowContext(ctx, `SELECT leaving, decided_at FROM departures WHERE account_id = $1`, uuid.UUID(account)).
		Scan(&kept.Leaving, &kept.DecidedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return departures.Snapshot{}, nil
	case err != nil:
		return departures.Snapshot{}, fmt.Errorf("failed to read the departure: %w", err)
	}
	kept.DecidedAt = kept.DecidedAt.UTC()
	return kept, nil
}
