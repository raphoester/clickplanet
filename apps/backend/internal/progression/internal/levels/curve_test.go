package levels_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
)

func TestTheXPToReachEachLevelIsTheDesignsTable(t *testing.T) {
	for _, c := range []struct {
		level         levels.Level
		total, toNext levels.XP
	}{
		{level: 1, total: 0, toNext: 14},
		{level: 2, total: 14, toNext: 98},
		{level: 3, total: 112, toNext: 266},
		{level: 4, total: 378, toNext: 518},
		{level: 5, total: 896, toNext: 854},
		{level: 6, total: 1_750, toNext: 1_274},
		{level: 8, total: 4_802, toNext: 2_366},
		{level: 10, total: 10_206, toNext: 3_794},
		{level: 12, total: 18_634, toNext: 5_558},
		{level: 15, total: 38_416, toNext: 8_834},
		{level: 20, total: 96_026, toNext: 15_974},
		{level: 30, total: 341_446, toNext: 36_554},
		{level: 40, total: 830_466, toNext: 65_534},
		{level: 50, total: 1_647_086, toNext: 102_914},
	} {
		t.Run(fmt.Sprintf("level %d", c.level), func(t *testing.T) {
			curve := levels.Curve{}

			assert.Equal(t, c.total, curve.Threshold(c.level))
			assert.Equal(t, c.toNext, curve.Threshold(c.level+1)-curve.Threshold(c.level))
		})
	}
}

func TestALevelIsReachedAtItsThresholdAndNotOneXPBefore(t *testing.T) {
	for _, c := range []struct {
		xp    levels.XP
		level levels.Level
	}{
		{xp: 0, level: 1},
		{xp: 13, level: 1},
		{xp: 14, level: 2},
		{xp: 111, level: 2},
		{xp: 112, level: 3},
		{xp: 895, level: 4},
		{xp: 896, level: 5},
		{xp: 10_205, level: 9},
		{xp: 10_206, level: 10},
		{xp: 341_445, level: 29},
		{xp: 341_446, level: 30},
		{xp: 1_647_086, level: 50},
	} {
		t.Run(fmt.Sprintf("%d XP", c.xp), func(t *testing.T) {
			assert.Equal(t, c.level, levels.Curve{}.LevelOf(c.xp))
		})
	}
}

func TestEveryLevelStartsExactlyAtItsThreshold(t *testing.T) {
	curve := levels.Curve{}
	for level := levels.Level(2); level <= 100_000; level++ {
		threshold := curve.Threshold(level)
		if curve.LevelOf(threshold) != level || curve.LevelOf(threshold-1) != level-1 {
			assert.Failf(t, "the level does not start at its threshold",
				"level %d: %d XP reads %d, %d XP reads %d",
				level, threshold, curve.LevelOf(threshold), threshold-1, curve.LevelOf(threshold-1))
			return
		}
	}
}

func TestThereIsNoTopLevel(t *testing.T) {
	curve := levels.Curve{}
	most := levels.XP(math.MaxUint64)

	assert.Equal(t, levels.Level(1_096_303), curve.LevelOf(most), "the highest level an XP can hold")
	assert.Equal(t, levels.XP(18_446_698_726_432_138_512), curve.Threshold(1_096_303))
	assert.Equal(t, most, curve.Threshold(1_096_304), "a level past any XP costs the most an XP holds")
}

func TestAStandingIsTheLevelAndTheBarItsXPFills(t *testing.T) {
	standing := levels.Curve{}.StandingOf(1_000)

	assert.Equal(t, levels.Level(5), standing.Level())
	assert.Equal(t, levels.XP(1_000), standing.XP())
	assert.Equal(t, levels.XP(896), standing.LevelXP())
	assert.Equal(t, levels.XP(1_750), standing.NextLevelXP())
	assert.False(t, standing.Empty())
}
