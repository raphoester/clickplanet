package postgres_removal_store

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Querier) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Querier
}

var _ removals.Storage = (*Store)(nil)

func (s *Store) Save(ctx context.Context, removal removals.Snapshot) error {
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO removals (message_id, removed_at) VALUES ($1, $2)
		ON CONFLICT (message_id) DO UPDATE SET removed_at = $2
	`, string(removal.MessageID), removal.RemovedAt.UTC()); err != nil {
		return fmt.Errorf("failed to save a removal: %w", err)
	}
	return nil
}

func (s *Store) DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM removals WHERE removed_at < $1`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to prune the removals: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to count the removals pruned: %w", err)
	}
	return deleted, nil
}
