//go:build testing

package postgres_player_store

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/players"
)

func (s *Store) IsPlayer(ctx context.Context, account players.AccountID) (bool, error) {
	var is bool
	if err := s.db.On(ctx).QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM players WHERE account_id = $1)`, uuid.UUID(account)).Scan(&is); err != nil {
		return false, fmt.Errorf("failed to read the player: %w", err)
	}
	return is, nil
}
