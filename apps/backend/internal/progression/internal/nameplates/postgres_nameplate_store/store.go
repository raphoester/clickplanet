package postgres_nameplate_store

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/nameplates"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ nameplates.Store = (*Store)(nil)

func (s *Store) Keep(ctx context.Context, account accounts.AccountID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx,
		`INSERT INTO nameplates (account_id) VALUES ($1) ON CONFLICT DO NOTHING`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to keep the nameplate: %w", err)
	}
	return nil
}

func (s *Store) Forget(ctx context.Context, account accounts.AccountID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM nameplates WHERE account_id = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to forget the nameplate: %w", err)
	}
	return nil
}
