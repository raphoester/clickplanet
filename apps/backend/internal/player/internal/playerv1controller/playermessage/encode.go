// Package playermessage writes the domain's values as player.v1 messages, for the handlers that answer them.
package playermessage

import (
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

func Profile(profile players.Profile) *playerv1.Profile {
	return &playerv1.Profile{AccountId: profile.Account.String(), Name: string(profile.Name)}
}

func Stats(stats players.Stats) *playerv1.Stats {
	return &playerv1.Stats{
		TilesTaken:    stats.TilesTaken,
		StreakCurrent: stats.StreakCurrent,
		StreakBest:    stats.StreakBest,
		StreakLastDay: stats.StreakLastDay.String(),
	}
}
