package postgres_gift_store

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/gifts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Querier) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Querier
}

var _ gifts.Storage = (*Store)(nil)

func (s *Store) Give(ctx context.Context, tag tempo.GiftTag, holder bonuses.Holder) error {
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO gifts (tag, account) VALUES ($1, $2::uuid) ON CONFLICT DO NOTHING`, string(tag), string(holder))
	if err != nil {
		return fmt.Errorf("failed to record the %s gift: %w", tag, err)
	}

	inserted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read whether the %s gift was recorded: %w", tag, err)
	}
	if inserted == 0 {
		return gifts.ErrGiven
	}

	return nil
}
