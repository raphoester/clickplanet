package postgres_rally_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rallies"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

const rallyLock = 0x72616c6c

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ rallies.Store = (*Store)(nil)

func (s *Store) Last(ctx context.Context, account cpsession.AccountID) (rallies.Rally, error) {
	return lastOf(ctx, s.db.On(ctx), account)
}

func (s *Store) Change(
	ctx context.Context, account cpsession.AccountID, change func(rallies.Rally) (rallies.Rally, error),
) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1, hashtext($2))`, rallyLock, account.String()); err != nil {
			return fmt.Errorf("failed to lock the account's rally: %w", err)
		}

		last, err := lastOf(ctx, tx, account)
		if err != nil {
			return err
		}

		next, err := change(last)
		if err != nil {
			return err
		}

		kept := next.TakeSnapshot()
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO rallies (account, id, flag, x, y, z, radius, called_at, ends_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (account) DO UPDATE SET id = EXCLUDED.id, flag = EXCLUDED.flag, x = EXCLUDED.x, y = EXCLUDED.y,
				z = EXCLUDED.z, radius = EXCLUDED.radius, called_at = EXCLUDED.called_at, ends_at = EXCLUDED.ends_at
		`, account.String(), kept.ID.String(), kept.Flag, kept.Centre.X, kept.Centre.Y, kept.Centre.Z, kept.Radius,
			kept.CalledAt.UTC(), kept.EndsAt.UTC()); err != nil {
			return fmt.Errorf("failed to write the rally: %w", err)
		}
		return nil
	})
}

func (s *Store) On(ctx context.Context, flag string, at time.Time) ([]rallies.Rally, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT account, id, flag, x, y, z, radius, called_at, ends_at FROM rallies WHERE flag = $1 AND ends_at > $2
	`, flag, at.UTC())
	if err != nil {
		return nil, fmt.Errorf("failed to read the rallies on: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var on []rallies.Rally
	for rows.Next() {
		kept, err := scan(rows)
		if err != nil {
			return nil, err
		}
		on = append(on, kept.Restore())
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the rallies on: %w", err)
	}
	return on, nil
}

func lastOf(ctx context.Context, db cppg.Querier, account cpsession.AccountID) (rallies.Rally, error) {
	kept, err := scan(db.QueryRowContext(ctx,
		`SELECT account, id, flag, x, y, z, radius, called_at, ends_at FROM rallies WHERE account = $1`, account.String()))
	if errors.Is(err, sql.ErrNoRows) {
		return rallies.Rally{}, nil
	}
	if err != nil {
		return rallies.Rally{}, err
	}
	return kept.Restore(), nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scan(row scanner) (rallies.Snapshot, error) {
	var (
		account, id uuid.UUID
		kept        rallies.Snapshot
	)
	if err := row.Scan(&account, &id, &kept.Flag, &kept.Centre.X, &kept.Centre.Y, &kept.Centre.Z, &kept.Radius, &kept.CalledAt,
		&kept.EndsAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return rallies.Snapshot{}, err //nolint:wrapcheck // the caller tells no rally from a failure by it.
		}
		return rallies.Snapshot{}, fmt.Errorf("failed to read a rally: %w", err)
	}
	kept.Account, kept.ID = cpsession.AccountID(account), rallies.RallyID(id)
	return kept, nil
}
