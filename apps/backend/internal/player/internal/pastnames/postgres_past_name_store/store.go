package postgres_past_name_store

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/pastnames"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ pastnames.Storage = (*Store)(nil)

func (s *Store) Record(ctx context.Context, name pastnames.PastName) error {
	kept := name.TakeSnapshot()
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO past_names (account_id, name, name_folded, left_at) VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING
	`, uuid.UUID(kept.Account), string(kept.Name), kept.Name.Folded(), kept.LeftAt.UTC()); err != nil {
		return fmt.Errorf("failed to record a past name: %w", err)
	}
	return nil
}

func (s *Store) Forget(ctx context.Context, account players.AccountID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM past_names WHERE account_id = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to forget an account's past names: %w", err)
	}
	return nil
}
