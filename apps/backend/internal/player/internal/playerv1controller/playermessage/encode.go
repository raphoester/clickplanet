package playermessage

import (
	"fmt"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
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

func Title(standing titles.Standing) *playerv1.Title {
	if standing.Empty() {
		return nil
	}

	title := &playerv1.Title{Id: string(standing.Title.ID()), Name: standing.Title.Name()}
	if standing.Place.Ranked() {
		title.Rank = &playerv1.Rank{
			TrackId:   string(standing.Place.Track),
			TrackName: standing.Place.TrackName,
			Number:    uint32(standing.Place.Number), //nolint:gosec // a place in a track of a handful of ranks.
			Count:     uint32(standing.Place.Count),  //nolint:gosec // as above.
		}
	}
	return title
}

func Titles(standings []titles.Standing) []*playerv1.Title {
	encoded := make([]*playerv1.Title, 0, len(standings))
	for _, standing := range standings {
		encoded = append(encoded, Title(standing))
	}
	return encoded
}

func Dashboard(showcase wearing.Showcase, progress []titles.TrackProgress) *playerv1.GetTitlesResponse {
	tracks := make([]*playerv1.Track, 0, len(progress))
	for _, track := range progress {
		steps := make([]*playerv1.Step, 0, len(track.Steps))
		for _, step := range track.Steps {
			steps = append(steps, &playerv1.Step{Title: Title(step.Standing), Threshold: step.Threshold, Earned: step.Earned})
		}
		tracks = append(tracks, &playerv1.Track{Id: string(track.ID), Name: track.Name, Progress: track.Progress, Steps: steps})
	}
	return &playerv1.GetTitlesResponse{Worn: Title(showcase.Worn), Wearable: Titles(showcase.Shown), Tracks: tracks}
}

func Player(player players.Player, showcase wearing.Showcase) *playerv1.Player {
	message := &playerv1.Player{
		Name: string(player.Name), Stats: Stats(player.Stats), Admin: player.Admin, Color: Color(player.Color),
		Titles: Titles(showcase.Shown), WornTitle: Title(showcase.Worn),
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
