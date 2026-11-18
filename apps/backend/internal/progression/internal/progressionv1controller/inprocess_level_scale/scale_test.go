package inprocess_level_scale_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/inprocess_level_scale"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/progressionmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/progressionread"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
)

func TestEachSourceHasItsOwnFieldAndTheyAddUpToTheTotal(t *testing.T) {
	xp := inprocess_level_scale.New(levels.Scale{}).XP(progressionread.Record{
		TilesTaken:     600,
		DaysPlayed:     2,
		MissionGoals:   map[string]uint64{"take_for": 180, "catch_boxes": 4},
		BombsDropped:   3,
		BonusesClaimed: 4,
		RightAnswers:   2,
		TilesFortified: 1_234,
	})

	assert.True(t, proto.Equal(&progressionv1.XpBreakdown{
		Total:          600 + 200 + 380 + 150 + 120 + 60 + 246,
		TilesTaken:     600,
		DaysPlayed:     200,
		MissionsDone:   380,
		BombsDropped:   150,
		BoxesCaught:    120,
		RightAnswers:   60,
		TilesFortified: 246,
	}, xp), "got %v", xp)
}

func TestNothingDoneIsNoXP(t *testing.T) {
	assert.True(t, proto.Equal(&progressionv1.XpBreakdown{}, inprocess_level_scale.New(levels.Scale{}).XP(progressionread.Record{})))
}

var firstDay = progressionread.Record{TilesTaken: 600, DaysPlayed: 1, MissionGoals: map[string]uint64{"take_for": 180}}

func TestTheLevelKeptIsAnsweredOnItsBarWithTheXPOfNow(t *testing.T) {
	scale := inprocess_level_scale.New(levels.Scale{})

	assert.Nil(t, scale.Level(firstDay, 0), "none kept")
	assert.True(t, proto.Equal(&progressionv1.Level{Level: 3, Xp: 880, LevelXp: 112, NextLevelXp: 378}, scale.Level(firstDay, 3)))
	assert.True(t, proto.Equal(&progressionv1.Level{Level: 1, NextLevelXp: 14}, scale.Level(progressionread.Record{}, 1)))
}

func TestEveryKindOfMissionButUnspecifiedHasOneRateOnTheWire(t *testing.T) {
	rates, err := inprocess_level_scale.New(levels.Scale{}).MissionRates()
	require.NoError(t, err)

	told, rated := make([]missionsv1.MissionKind, 0, len(rates)), make([]missionsv1.MissionKind, 0, len(rates))
	for number := range missionsv1.MissionKind_name {
		if number != int32(missionsv1.MissionKind_MISSION_KIND_UNSPECIFIED) {
			told = append(told, missionsv1.MissionKind(number))
		}
	}
	for _, rate := range rates {
		rated = append(rated, rate.GetKind())
	}
	assert.ElementsMatch(t, told, rated)
}

func TestThePagesIntegerMathPaysWhatTheServerDoes(t *testing.T) {
	sent, err := inprocess_level_scale.New(levels.Scale{}).MissionRates()
	require.NoError(t, err)

	rates := levels.MissionsDone{}.Rates()
	require.Len(t, sent, len(rates))
	for n, rate := range rates {
		wire := sent[n]
		for goal := range uint64(1_000) {
			require.Equal(t, rate.Rate().XPOf(goal), levels.XP(goal*uint64(wire.GetXp())/uint64(wire.GetPer())),
				"%s, a goal of %d", wire.GetKind(), goal)
		}
	}
	assert.True(t, proto.Equal(
		&progressionv1.MissionRate{Kind: missionsv1.MissionKind_MISSION_KIND_TAKE_FROM, Xp: 3, Per: 2}, sent[1],
	), "1.5 a tile, sent as 3 for 2: %v", sent[1])
}

func TestEachRatedSourceSaysWhatItPaysOnTheWire(t *testing.T) {
	rates, err := inprocess_level_scale.New(levels.Scale{}).XPRates()
	require.NoError(t, err)

	assert.True(t, proto.Equal(&progressionv1.GetLevelResponse{XpRates: []*progressionv1.XpRate{
		{Source: progressionv1.XpSource_XP_SOURCE_TILES_TAKEN, Xp: 1, Per: 1},
		{Source: progressionv1.XpSource_XP_SOURCE_DAYS_PLAYED, Xp: 100, Per: 1},
		{Source: progressionv1.XpSource_XP_SOURCE_BOMBS_DROPPED, Xp: 50, Per: 1},
		{Source: progressionv1.XpSource_XP_SOURCE_BOXES_CAUGHT, Xp: 30, Per: 1},
		{Source: progressionv1.XpSource_XP_SOURCE_RIGHT_ANSWERS, Xp: 30, Per: 1},
		{Source: progressionv1.XpSource_XP_SOURCE_TILES_FORTIFIED, Xp: 1, Per: 5},
	}}, &progressionv1.GetLevelResponse{XpRates: rates}), "got %v", rates)
}

// A gain whose source the wire does not name would never reach the page.
func TestEverySourceThatGivesXPIsNamedOnTheWire(t *testing.T) {
	everything := stats.StatsSnapshot{
		TilesTaken: 100, DaysPlayed: 100, BombsDropped: 100, BonusesClaimed: 100, RightAnswers: 100, TilesFortified: 100,
		MessagesSent:   100,
		MissionsByKind: map[stats.MissionKind]stats.MissionTally{"take_for": {Done: 1, Goals: 100}},
	}.Restore()

	named := []progressionv1.XpSource{}
	for _, source := range (levels.Scale{}).Sources() {
		if source.XP(everything) == 0 {
			continue
		}
		wire, ok := progressionmessage.XPSource(source.Name())
		require.True(t, ok, "%q has no XpSource", source.Name())
		named = append(named, wire)
	}

	all := []progressionv1.XpSource{}
	for number := range progressionv1.XpSource_name {
		if number != int32(progressionv1.XpSource_XP_SOURCE_UNSPECIFIED) {
			all = append(all, progressionv1.XpSource(number))
		}
	}
	assert.ElementsMatch(t, all, named, "every XpSource is a source that gives XP")
}
