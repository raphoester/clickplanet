package players_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

func TestEachTitleIsEarnedAtItsThresholdAndNotBefore(t *testing.T) {
	for _, c := range []struct {
		title         players.Title
		before, after players.Stats
	}{
		{players.Settler{}, players.Stats{TilesTaken: 99}, players.Stats{TilesTaken: 100}},
		{players.Governor{}, players.Stats{TilesTaken: 999}, players.Stats{TilesTaken: 1_000}},
		{players.Conqueror{}, players.Stats{TilesTaken: 9_999}, players.Stats{TilesTaken: 10_000}},
		{players.Emperor{}, players.Stats{TilesTaken: 99_999}, players.Stats{TilesTaken: 100_000}},
		{players.Loyal{}, players.Stats{StreakBest: 6}, players.Stats{StreakBest: 7}},
		{players.Devoted{}, players.Stats{StreakBest: 29}, players.Stats{StreakBest: 30}},
		{players.Unbroken{}, players.Stats{StreakBest: 99}, players.Stats{StreakBest: 100}},
	} {
		assert.False(t, c.title.EarnedBy(c.before), "%s at %+v", c.title.ID(), c.before)
		assert.True(t, c.title.EarnedBy(c.after), "%s at %+v", c.title.ID(), c.after)
	}
}

func TestAStreakTitleCountsTheBestStreakNotTheCurrentOne(t *testing.T) {
	assert.False(t, players.Devoted{}.EarnedBy(players.Stats{StreakCurrent: 30, StreakBest: 1}))
	assert.True(t, players.Devoted{}.EarnedBy(players.Stats{StreakCurrent: 0, StreakBest: 30}))
}

func TestEveryTitleHasItsOwnIDTheStoreTakesAndAName(t *testing.T) {
	pattern := regexp.MustCompile(`^[a-z_]+$`)
	seen := cpcolls.NewSet[players.TitleID]()

	for _, title := range players.NewCatalog() {
		assert.Regexp(t, pattern, string(title.ID()))
		assert.NotEmpty(t, title.Name(), title.ID())
		assert.False(t, seen.Contains(title.ID()), "%s is listed twice", title.ID())
		seen.Add(title.ID())
	}
}
