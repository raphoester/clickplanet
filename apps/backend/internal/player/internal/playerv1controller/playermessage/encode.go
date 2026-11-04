package playermessage

import (
	"fmt"
	"time"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
)

func Profile(profile players.Profile) *playerv1.Profile {
	kept := profile.TakeSnapshot()
	return &playerv1.Profile{
		AccountId: kept.Account.String(), Name: string(kept.Name), RenamableAtUnixMs: RenamableAt(kept.ChosenAt),
	}
}

func RenamableAt(chosenAt time.Time) int64 {
	at := players.NameChosenAt(chosenAt).RenamableAt()
	if at.IsZero() {
		return 0
	}
	return at.UnixMilli()
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

func RosterEntry(entry presence.Entry) *playerv1.RosterEntry {
	return &playerv1.RosterEntry{
		Key:       string(entry.Key()),
		AccountId: entry.Account().String(),
		Name:      entry.Name(),
		CountryId: entry.Country(),
		Guest:     entry.Guest(),
		Admin:     entry.Admin(),
		Color:     Color(entry.Color()),
	}
}

func RosterEntries(roster []presence.Entry) []*playerv1.RosterEntry {
	entries := make([]*playerv1.RosterEntry, 0, len(roster))
	for _, entry := range roster {
		entries = append(entries, RosterEntry(entry))
	}
	return entries
}
