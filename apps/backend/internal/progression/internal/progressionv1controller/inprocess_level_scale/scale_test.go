package inprocess_level_scale_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

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
