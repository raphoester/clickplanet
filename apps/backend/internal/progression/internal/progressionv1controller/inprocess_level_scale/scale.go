package inprocess_level_scale

import (
	"errors"
	"fmt"
	"strings"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/planetwords"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/progressionmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/progressionread"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
)

var ErrUnknownKind = errors.New("a kind of mission the wire does not name")

func New(scale levels.Scale) Scale {
	return Scale{scale: scale}
}

type Scale struct {
	scale levels.Scale
}

func (s Scale) XP(read progressionread.Record) *progressionv1.XpBreakdown {
	record := recordOf(read)
	return &progressionv1.XpBreakdown{
		Total:          uint64(s.scale.XPOf(record)),
		TilesTaken:     uint64(levels.TilesTaken{}.XP(record)),
		DaysPlayed:     uint64(levels.DaysPlayed{}.XP(record)),
		MissionsDone:   uint64(levels.MissionsDone{}.XP(record)),
		BombsDropped:   uint64(levels.BombsDropped{}.XP(record)),
		BoxesCaught:    uint64(levels.BoxesCaught{}.XP(record)),
		RightAnswers:   uint64(levels.RightAnswers{}.XP(record)),
		TilesFortified: uint64(levels.TilesFortified{}.XP(record)),
	}
}

func (s Scale) Level(read progressionread.Record, kept uint32) *progressionv1.Level {
	return progressionmessage.Level(s.scale.StandingAt(levels.Level(kept), recordOf(read)))
}

func (s Scale) MissionRates() ([]*progressionv1.MissionRate, error) {
	rates := levels.MissionsDone{}.Rates()
	encoded := make([]*progressionv1.MissionRate, 0, len(rates))
	for _, rate := range rates {
		kind, known := missionsv1.MissionKind_value["MISSION_KIND_"+strings.ToUpper(string(rate.Kind()))]
		if !known {
			return nil, fmt.Errorf("%w: %q", ErrUnknownKind, rate.Kind())
		}
		encoded = append(encoded, &progressionv1.MissionRate{
			Kind: missionsv1.MissionKind(kind),
			Xp:   uint32(rate.Rate().XP),  //nolint:gosec // a few hundred XP at most.
			Per:  uint32(rate.Rate().Per), //nolint:gosec // as above.
		})
	}
	return encoded, nil
}

// Unlocks is the ladder in planet's words: each kind a level unlocks, at its level.
func (s Scale) Unlocks() ([]*progressionv1.Unlock, error) {
	ladder := levels.Ladder{}.Unlocks()
	encoded := make([]*progressionv1.Unlock, 0, len(ladder))
	for _, unlock := range ladder {
		kind, err := planetwords.BonusKind(unlock.Kind())
		if err != nil {
			return nil, err //nolint:wrapcheck // the sentinel names the kind.
		}
		encoded = append(encoded, &progressionv1.Unlock{Kind: kind, Level: uint32(unlock.Level())})
	}
	return encoded, nil
}

func recordOf(read progressionread.Record) stats.Stats {
	byKind := make(map[stats.MissionKind]stats.MissionTally, len(read.MissionGoals))
	for kind, goals := range read.MissionGoals {
		byKind[stats.MissionKind(kind)] = stats.MissionTally{Goals: goals}
	}
	return stats.StatsSnapshot{
		TilesTaken:     read.TilesTaken,
		DaysPlayed:     read.DaysPlayed,
		MissionsByKind: byKind,
		BombsDropped:   read.BombsDropped,
		BonusesClaimed: read.BonusesClaimed,
		RightAnswers:   read.RightAnswers,
		TilesFortified: read.TilesFortified,
	}.Restore()
}
