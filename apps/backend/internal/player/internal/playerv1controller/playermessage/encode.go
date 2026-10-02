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

var titles = map[players.Title]playerv1.Title{
	players.Settler:   playerv1.Title_TITLE_SETTLER,
	players.Governor:  playerv1.Title_TITLE_GOVERNOR,
	players.Conqueror: playerv1.Title_TITLE_CONQUEROR,
	players.Emperor:   playerv1.Title_TITLE_EMPEROR,
	players.Loyal:     playerv1.Title_TITLE_LOYAL,
	players.Devoted:   playerv1.Title_TITLE_DEVOTED,
	players.Unbroken:  playerv1.Title_TITLE_UNBROKEN,
}

func Titles(held players.Titles) []playerv1.Title {
	encoded := make([]playerv1.Title, 0, len(held))
	for _, title := range held {
		if wire, ok := titles[title]; ok {
			encoded = append(encoded, wire)
		}
	}
	return encoded
}

func Player(player players.Player) *playerv1.Player {
	message := &playerv1.Player{
		Name: string(player.Name), Stats: Stats(player.Stats), Admin: player.Admin, Titles: Titles(player.Titles),
	}
	if !player.CreatedAt.IsZero() {
		message.CreatedAtUnixMs = player.CreatedAt.UnixMilli()
	}
	return message
}

func RosterEntry(entry presence.Entry) *playerv1.RosterEntry {
	return &playerv1.RosterEntry{
		Key:       string(entry.Key),
		Name:      entry.Name,
		CountryId: entry.Country,
		Guest:     entry.Guest,
		Admin:     entry.Admin,
	}
}

func RosterEntries(roster []presence.Entry) []*playerv1.RosterEntry {
	entries := make([]*playerv1.RosterEntry, 0, len(roster))
	for _, entry := range roster {
		entries = append(entries, RosterEntry(entry))
	}
	return entries
}
