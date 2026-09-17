// Package get_stats_usecase reads the caller's stats, as of today.
package get_stats_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Stats interface {
	Stats(ctx context.Context, account players.AccountID) (players.Stats, error)
}

type UseCase struct {
	stats Stats
	clock cptime.Clock
}

func New(stats Stats, clock cptime.Clock) *UseCase {
	return &UseCase{stats: stats, clock: clock}
}

// Execute answers empty stats for an account that never took a tile, and a streak over once a whole UTC day went by.
func (u *UseCase) Execute(ctx context.Context, account players.AccountID) (players.Stats, error) {
	stats, err := u.stats.Stats(ctx, account)
	if errors.Is(err, players.ErrNoStats) {
		return players.Stats{Account: account}, nil
	}
	if err != nil {
		return players.Stats{}, fmt.Errorf("failed to read the stats: %w", err)
	}
	return stats.AsOf(players.DayOf(u.clock.Now())), nil
}
