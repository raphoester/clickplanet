package postgres_arrival_store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ arrivals.Store = (*Store)(nil)

func (s *Store) Save(ctx context.Context, saved ...arrivals.Arrival) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		for _, arrival := range saved {
			kept := arrival.TakeSnapshot()
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO arrivals (id, account_id, source, flag, arrived_at)
				VALUES ($1, $2, $3, $4, $5)
				ON CONFLICT (id) DO UPDATE SET
					account_id = EXCLUDED.account_id, source = EXCLUDED.source, flag = EXCLUDED.flag,
					arrived_at = EXCLUDED.arrived_at
			`,
				uuid.UUID(kept.ID), holder(kept.Account), string(kept.Source), kept.Flag, kept.At.UTC(),
			); err != nil {
				return fmt.Errorf("failed to upsert an arrival: %w", err)
			}
		}
		return nil
	})
}

// An anonymized arrival has no account: NULL.
func holder(account arrivals.AccountID) uuid.NullUUID {
	return uuid.NullUUID{UUID: uuid.UUID(account), Valid: account != arrivals.AccountID{}}
}

func (s *Store) Of(ctx context.Context, account arrivals.AccountID) ([]arrivals.Arrival, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT id, account_id, source, flag, arrived_at
		FROM arrivals
		WHERE account_id = $1
		ORDER BY arrived_at, id
	`, uuid.UUID(account))
	if err != nil {
		return nil, fmt.Errorf("failed to read the arrivals: %w", err)
	}
	defer func() { _ = rows.Close() }()

	found := []arrivals.Arrival{}
	for rows.Next() {
		var (
			id      uuid.UUID
			account uuid.NullUUID
			source  string
			flag    string
			at      time.Time
		)
		if err := rows.Scan(&id, &account, &source, &flag, &at); err != nil {
			return nil, fmt.Errorf("failed to read an arrival: %w", err)
		}
		arrival, err := arrivals.Snapshot{
			ID: arrivals.ArrivalID(id), Account: arrivals.AccountID(account.UUID), Source: arrivals.Source(source),
			Flag: flag, At: at.UTC(),
		}.Restore()
		if err != nil {
			return nil, fmt.Errorf("failed to restore the arrival %s: %w", id, err)
		}
		found = append(found, arrival)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the arrivals: %w", err)
	}
	return found, nil
}
