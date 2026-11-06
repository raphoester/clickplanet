package postgres_player_store

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ players.Store = (*Store)(nil)

func (s *Store) Keep(ctx context.Context, account players.AccountID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx,
		`INSERT INTO players (account_id) VALUES ($1) ON CONFLICT DO NOTHING`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to keep the player: %w", err)
	}
	return nil
}

func (s *Store) IsPlayer(ctx context.Context, account players.AccountID) (bool, error) {
	var is bool
	if err := s.db.On(ctx).QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM players WHERE account_id = $1)`, uuid.UUID(account)).Scan(&is); err != nil {
		return false, fmt.Errorf("failed to read the player: %w", err)
	}
	return is, nil
}

func (s *Store) Forget(ctx context.Context, account players.AccountID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM players WHERE account_id = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to forget the player: %w", err)
	}
	return nil
}
