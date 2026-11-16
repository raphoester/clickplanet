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
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/progressionread"
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
