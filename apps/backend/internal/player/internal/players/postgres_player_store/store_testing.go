//go:build testing

package postgres_player_store

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

const upsertTakes = upsertStats + `tiles_taken = excluded.tiles_taken, streak_current = excluded.streak_current,
	streak_best = excluded.streak_best, streak_last_day = excluded.streak_last_day`

func (s *Store) RecordTake(ctx context.Context, account players.AccountID, at time.Time) error {
	return s.record(ctx, account, func(stats players.Stats) players.Stats { return stats.WithTake(at) }, upsertTakes)
}
