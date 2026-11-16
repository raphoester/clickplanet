package inprocess_level_scale

import (
	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/progressionread"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
)

func New(scale levels.Scale) Scale {
	return Scale{scale: scale}
}

type Scale struct {
	scale levels.Scale
}

func (s Scale) XP(read progressionread.Record) *progressionv1.XpBreakdown {
	byKind := make(map[stats.MissionKind]stats.MissionTally, len(read.MissionGoals))
	for kind, goals := range read.MissionGoals {
		byKind[stats.MissionKind(kind)] = stats.MissionTally{Goals: goals}
	}
	record := stats.StatsSnapshot{
		TilesTaken:     read.TilesTaken,
		DaysPlayed:     read.DaysPlayed,
		MissionsByKind: byKind,
		BombsDropped:   read.BombsDropped,
		BonusesClaimed: read.BonusesClaimed,
		RightAnswers:   read.RightAnswers,
		TilesFortified: read.TilesFortified,
	}.Restore()

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
