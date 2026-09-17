// Package get_stats_usecase reads the caller's stats, as of today.
package get_stats_usecase

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Stats interface {
	Stats(account players.AccountID) (players.Stats, bool)
}

type UseCase struct {
	stats Stats
	clock cptime.Clock
}

func New(stats Stats, clock cptime.Clock) *UseCase {
	return &UseCase{stats: stats, clock: clock}
}

// Execute answers empty stats for an account that never took a tile, and a streak over once a whole UTC day went by.
func (u *UseCase) Execute(account players.AccountID) players.Stats {
	stats, ok := u.stats.Stats(account)
	if !ok {
		return players.Stats{Account: account}
	}
	return stats.AsOf(players.DayOf(u.clock.Now()))
}
