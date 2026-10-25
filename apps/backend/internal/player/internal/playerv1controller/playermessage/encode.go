package playermessage

import (
	"fmt"
	"time"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
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

func Title(standing titles.Standing) *playerv1.Title {
	if standing.Empty() {
		return nil
	}

	title := &playerv1.Title{Id: string(standing.Title().ID()), Name: standing.Title().Name()}
	if place := standing.Place(); place.Ranked() {
		title.Rank = &playerv1.Rank{
			TrackId:   string(place.Track()),
			TrackName: place.TrackName(),
			Number:    uint32(place.Number()), //nolint:gosec // a place in a track of a handful of ranks.
			Count:     uint32(place.Count()),  //nolint:gosec // as above.
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

func RosterEntry(entry presence.Entry) *playerv1.RosterEntry {
	return &playerv1.RosterEntry{
		Key:       string(entry.Key()),
		AccountId: entry.Account().String(),
		Name:      entry.Name(),
		CountryId: entry.Country(),
		Guest:     entry.Guest(),
		Admin:     entry.Admin(),
		Color:     Color(entry.Color()),
		Streak:    entry.Streak(),
		WornTitle: Title(entry.Title()),
	}
}

func RosterEntries(roster []presence.Entry) []*playerv1.RosterEntry {
	entries := make([]*playerv1.RosterEntry, 0, len(roster))
	for _, entry := range roster {
		entries = append(entries, RosterEntry(entry))
	}
	return entries
}

func Mission(mission missions.MissionSnapshot, sheet missions.SheetSnapshot) *playerv1.Mission {
	return &playerv1.Mission{
		Slot:            uint32(mission.Slot),
		Kind:            MissionKind(mission.Kind),
		TargetCountryId: mission.Target,
		Goal:            mission.Goal,
		Progress:        sheet.Taken[mission.Slot],
		Collected:       sheet.Collected.Contains(mission.Slot),
		RewardKind:      RewardKind(mission.Reward.Kind),
		RewardAmount:    mission.Reward.Amount,
	}
}

func MissionKind(kind missions.Kind) playerv1.MissionKind {
	switch kind {
	case missions.KindTakeFor:
		return playerv1.MissionKind_MISSION_KIND_TAKE_FOR
	case missions.KindTakeFrom:
		return playerv1.MissionKind_MISSION_KIND_TAKE_FROM
	case missions.KindTakeIn:
		return playerv1.MissionKind_MISSION_KIND_TAKE_IN
	}
	return playerv1.MissionKind_MISSION_KIND_UNSPECIFIED
}

func RewardKind(kind missions.RewardKind) planetv1.BonusKind {
	switch kind {
	case missions.RewardShields:
		return planetv1.BonusKind_BONUS_KIND_SHIELDS
	case missions.RewardSpreadClicks:
		return planetv1.BonusKind_BONUS_KIND_SPREAD_CLICKS
	case missions.RewardBomb:
		return planetv1.BonusKind_BONUS_KIND_BOMB
	}
	return planetv1.BonusKind_BONUS_KIND_UNSPECIFIED
}
