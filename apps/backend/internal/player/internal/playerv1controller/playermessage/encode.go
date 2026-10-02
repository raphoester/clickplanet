package playermessage

import (
	"fmt"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
)

func Profile(profile players.Profile) *playerv1.Profile {
	return &playerv1.Profile{AccountId: profile.Account.String(), Name: string(profile.Name)}
}

func Color(color players.Color) playerv1.NameColor {
	return playerv1.NameColor(color)
}

func ColorOf(color playerv1.NameColor) (players.Color, error) {
	if _, named := playerv1.NameColor_name[int32(color)]; !named {
		return 0, fmt.Errorf("%w: %d", players.ErrInvalidColor, color)
	}
	return players.Color(color), nil
}

func Stats(stats players.Stats) *playerv1.Stats {
	return &playerv1.Stats{
		TilesTaken:    stats.TilesTaken,
		StreakCurrent: stats.StreakCurrent,
		StreakBest:    stats.StreakBest,
		StreakLastDay: stats.StreakLastDay.String(),
	}
}

func Titles(titles []players.Title) []*playerv1.Title {
	encoded := make([]*playerv1.Title, 0, len(titles))
	for _, title := range titles {
		encoded = append(encoded, &playerv1.Title{Id: string(title.ID()), Name: title.Name()})
	}
	return encoded
}

func Player(player players.Player) *playerv1.Player {
	message := &playerv1.Player{
		Name: string(player.Name), Stats: Stats(player.Stats), Admin: player.Admin, Color: Color(player.Color),
		Titles: Titles(player.Titles),
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
		Color:     Color(entry.Color),
		Streak:    entry.Streak,
	}
}

func RosterEntries(roster []presence.Entry) []*playerv1.RosterEntry {
	entries := make([]*playerv1.RosterEntry, 0, len(roster))
	for _, entry := range roster {
		entries = append(entries, RosterEntry(entry))
	}
	return entries
}
