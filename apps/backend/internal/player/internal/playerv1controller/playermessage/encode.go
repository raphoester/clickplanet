// Package playermessage writes the domain's values as player.v1 messages, for the handlers that answer them.
package playermessage

import (
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
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

// Player leaves out the account id: anybody may read it.
func Player(player players.Player) *playerv1.Player {
	message := &playerv1.Player{Name: string(player.Name), Stats: Stats(player.Stats)}
	if !player.CreatedAt.IsZero() {
		message.CreatedAtUnixMs = player.CreatedAt.UnixMilli()
	}
	return message
}

func RosterEntry(entry presence.Entry) *playerv1.RosterEntry {
	return &playerv1.RosterEntry{
		Key:       string(entry.Key),
		Name:      entry.Name,
		Tag:       string(entry.Tag),
		CountryId: entry.Country,
		Guest:     entry.Guest,
	}
}

func RosterEntries(roster []presence.Entry) []*playerv1.RosterEntry {
	entries := make([]*playerv1.RosterEntry, 0, len(roster))
	for _, entry := range roster {
		entries = append(entries, RosterEntry(entry))
	}
	return entries
}
