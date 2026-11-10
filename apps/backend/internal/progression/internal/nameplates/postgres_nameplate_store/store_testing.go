//go:build testing

package postgres_nameplate_store

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
)

func (s *Store) Named(ctx context.Context, account accounts.AccountID) (bool, error) {
	var named bool
	if err := s.db.On(ctx).QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM nameplates WHERE account_id = $1)`, uuid.UUID(account)).Scan(&named); err != nil {
		return false, fmt.Errorf("failed to read the nameplate: %w", err)
	}
	return named, nil
}
