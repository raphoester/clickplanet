package postgres_grant_store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

func (s *Store) Record(ctx context.Context, grant bonuses.GrantID, at time.Time) (bool, error) {
	kept, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO charge_grants (grant_id, granted_at) VALUES ($1, $2) ON CONFLICT (grant_id) DO NOTHING
	`, uuid.UUID(grant), at)
	if err != nil {
		return false, fmt.Errorf("failed to record grant %s: %w", grant, err)
	}
	rows, err := kept.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to read whether grant %s was recorded: %w", grant, err)
	}
	return rows == 1, nil
}
