package postgres_unlock_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/unlocks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const unlocksLock = 0x756e6c6b

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ unlocks.Store = (*Store)(nil)

func (s *Store) Change(ctx context.Context, account bonuses.Account, change func(unlocks.Unlocks) unlocks.Unlocks) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1, hashtext($2))`, unlocksLock, string(account)); err != nil {
			return fmt.Errorf("failed to lock the unlocks: %w", err)
		}

		id, err := idOf(account)
		if err != nil {
			return err
		}
		before, err := keptOf(ctx, tx, id)
		if err != nil {
			return err
		}
		after := change(before.Restore()).TakeSnapshot()
		if after.DecidedAt.Equal(before.DecidedAt) {
			return nil
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO unlocks (account_id, kinds, decided_at) VALUES ($1, $2::unlocked_kind[], $3)
			ON CONFLICT (account_id) DO UPDATE SET kinds = EXCLUDED.kinds, decided_at = EXCLUDED.decided_at
		`, id, namesOf(after.Kinds), after.DecidedAt.UTC()); err != nil {
			return fmt.Errorf("failed to write the unlocks: %w", err)
		}
		return nil
	})
}

func (s *Store) Unlocked(ctx context.Context, account bonuses.Account) (bonuses.Unlocked, error) {
	id, err := idOf(account)
	if err != nil {
		return nil, err
	}
	kept, err := keptOf(ctx, s.db.On(ctx), id)
	if err != nil {
		return nil, err
	}
	return kept.Kinds, nil
}

func (s *Store) Forget(ctx context.Context, account bonuses.Account) error {
	id, err := idOf(account)
	if err != nil {
		return err
	}
	if _, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM unlocks WHERE account_id = $1`, id); err != nil {
		return fmt.Errorf("failed to forget the unlocks: %w", err)
	}
	return nil
}

func idOf(account bonuses.Account) (uuid.UUID, error) {
	id, err := uuid.Parse(string(account))
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: %q", bonuses.ErrInvalidAccount, account)
	}
	return id, nil
}

func keptOf(ctx context.Context, db cppg.Querier, id uuid.UUID) (unlocks.Snapshot, error) {
	var (
		kept  unlocks.Snapshot
		names pq.StringArray
	)
	err := db.QueryRowContext(ctx, `SELECT kinds::text[], decided_at FROM unlocks WHERE account_id = $1`, id).
		Scan(&names, &kept.DecidedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return unlocks.Snapshot{}, nil
	case err != nil:
		return unlocks.Snapshot{}, fmt.Errorf("failed to read the unlocks: %w", err)
	}
	for _, name := range names {
		kept.Kinds = append(kept.Kinds, bonuses.Kind(name))
	}
	kept.DecidedAt = kept.DecidedAt.UTC()
	return kept, nil
}

func namesOf(kinds bonuses.Unlocked) pq.StringArray {
	names := make(pq.StringArray, 0, len(kinds))
	for _, kind := range kinds {
		names = append(names, string(kind))
	}
	return names
}
