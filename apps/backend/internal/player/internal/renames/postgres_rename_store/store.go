package postgres_rename_store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/renames"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ renames.Storage = (*Store)(nil)

func (s *Store) Record(ctx context.Context, rename renames.Rename) error {
	kept := rename.TakeSnapshot()
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO renames (account_id, renamed_at, old_name, new_name, reason, operator_id, client)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, uuid.UUID(kept.Account), kept.At.UTC(), string(kept.Old), string(kept.New), string(kept.Reason),
		operator(kept.By.Account()), kept.By.Client()); err != nil {
		return fmt.Errorf("failed to record a rename: %w", err)
	}
	return nil
}

func (s *Store) DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM renames WHERE renamed_at < $1`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to prune the renames: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to count the renames pruned: %w", err)
	}
	return deleted, nil
}

func (s *Store) Forget(ctx context.Context, account players.AccountID) error {
	return s.db.InTx(ctx, func(ctx context.Context) error { //nolint:wrapcheck // each statement names what failed.
		tx := s.db.On(ctx)
		if _, err := tx.ExecContext(ctx, `DELETE FROM renames WHERE account_id = $1`, uuid.UUID(account)); err != nil {
			return fmt.Errorf("failed to forget an account's renames: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE renames SET operator_id = NULL WHERE operator_id = $1
		`, uuid.UUID(account)); err != nil {
			return fmt.Errorf("failed to take an operator off its renames: %w", err)
		}
		return nil
	})
}

func operator(account players.AccountID) uuid.NullUUID {
	if account == cpsession.NoAccount {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: uuid.UUID(account), Valid: true}
}
