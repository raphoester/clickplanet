package progressionmessage

import (
	"strings"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles"
)

func Title(standing titles.Standing) *progressionv1.Title {
	if standing.Empty() {
		return nil
	}

	title := &progressionv1.Title{Id: string(standing.Title().ID()), Name: standing.Title().Name(), Feat: standing.Feat()}
	if place := standing.Place(); place.Ranked() {
		title.Rank = &progressionv1.Rank{
			TrackId:   string(place.Track()),
			TrackName: place.TrackName(),
			Number:    uint32(place.Number()), //nolint:gosec // a place in a track of a handful of ranks.
			Count:     uint32(place.Count()),  //nolint:gosec // as above.
		}
	}
	return title
}

func Titles(standings []titles.Standing) []*progressionv1.Title {
	encoded := make([]*progressionv1.Title, 0, len(standings))
	for _, standing := range standings {
		encoded = append(encoded, Title(standing))
	}
	return encoded
}

func Level(standing levels.Standing) *progressionv1.Level {
	if standing.Empty() {
		return nil
	}
	return &progressionv1.Level{
		Level:       uint32(standing.Level()),
		Xp:          uint64(standing.XP()),
		LevelXp:     uint64(standing.LevelXP()),
		NextLevelXp: uint64(standing.NextLevelXP()),
	}
}

// XPSource is the source in the wire's words: its name in upper case, after XP_SOURCE_. A source the wire does not name
// is not one.
func XPSource(source levels.SourceName) (progressionv1.XpSource, bool) {
	value, known := progressionv1.XpSource_value["XP_SOURCE_"+strings.ToUpper(string(source))]
	if !known || value == int32(progressionv1.XpSource_XP_SOURCE_UNSPECIFIED) {
		return progressionv1.XpSource_XP_SOURCE_UNSPECIFIED, false
	}
	return progressionv1.XpSource(value), true
}

// XPGained leaves out a gain whose source the wire does not name: TestEverySourceThatGivesXPIsNamedOnTheWire keeps
// there being none.
func XPGained(gained levels.Gained) *progressionv1.XpGained {
	encoded := &progressionv1.XpGained{Level: Level(gained.Standing)}
	for _, gain := range gained.Gains {
		if source, named := XPSource(gain.Source); named {
			encoded.Gains = append(encoded.Gains, &progressionv1.XpGain{Source: source, Xp: uint64(gain.XP)})
		}
	}
	return encoded
}
