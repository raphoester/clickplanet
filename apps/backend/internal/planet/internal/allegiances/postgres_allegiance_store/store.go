package postgres_allegiance_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const allegianceLock = 0x616c6c65

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

var _ allegiances.Store = (*Store)(nil)

func (s *Store) Allegiance(ctx context.Context, account allegiances.Account) (allegiances.Allegiance, error) {
	kept, err := keptOf(ctx, s.db, account)
	if err != nil {
		return allegiances.Allegiance{}, err
	}
	return kept.Restore(), nil
}

func (s *Store) Change(
	ctx context.Context,
	account allegiances.Account,
	change func(allegiances.Allegiance) (allegiances.Allegiance, error),
) (err error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin changing the allegiance: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1, hashtext($2))`, allegianceLock, string(account)); err != nil {
		return fmt.Errorf("failed to lock the allegiance: %w", err)
	}

	before, err := keptOf(ctx, tx, account)
	if err != nil {
		return err
	}

	after, err := change(before.Restore())
	if err != nil {
		return err
	}

	if kept := after.TakeSnapshot(); kept != before {
		changedAt := sql.NullTime{Time: kept.ChangedAt.UTC(), Valid: !kept.ChangedAt.IsZero()}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO allegiances (account, flag, changed_at) VALUES ($1, $2, $3)
			ON CONFLICT (account) DO UPDATE SET flag = EXCLUDED.flag, changed_at = EXCLUDED.changed_at
		`, string(account), kept.Flag, changedAt); err != nil {
			return fmt.Errorf("failed to write the allegiance: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the allegiance: %w", err)
	}
	return nil
}

func keptOf(ctx context.Context, db cppg.Querier, account allegiances.Account) (allegiances.Snapshot, error) {
	var (
		flag      string
		changedAt sql.NullTime
	)
	err := db.QueryRowContext(ctx, `SELECT flag, changed_at FROM allegiances WHERE account = $1`, string(account)).
		Scan(&flag, &changedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return allegiances.Snapshot{}, nil
	case err != nil:
		return allegiances.Snapshot{}, fmt.Errorf("failed to read the allegiance: %w", err)
	}

	return allegiances.Snapshot{Flag: flag, ChangedAt: changedAt.Time}, nil
}
