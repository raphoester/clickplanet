package players_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

func TestATitleIsEarnedAtItsThresholdAndNotBefore(t *testing.T) {
	for _, c := range []struct {
		stats players.Stats
		want  players.Titles
	}{
		{players.Stats{TilesTaken: 99, StreakBest: 6}, nil},
		{players.Stats{TilesTaken: 100}, players.Titles{players.Settler}},
		{players.Stats{TilesTaken: 9_999}, players.Titles{players.Settler, players.Governor}},
		{players.Stats{TilesTaken: 10_000}, players.Titles{players.Settler, players.Governor, players.Conqueror}},
		{players.Stats{TilesTaken: 100_000}, players.Titles{players.Settler, players.Governor, players.Conqueror, players.Emperor}},
		{players.Stats{StreakBest: 7}, players.Titles{players.Loyal}},
		{players.Stats{StreakBest: 30}, players.Titles{players.Loyal, players.Devoted}},
		{players.Stats{StreakBest: 100}, players.Titles{players.Loyal, players.Devoted, players.Unbroken}},
		{players.Stats{TilesTaken: 150, StreakBest: 8}, players.Titles{players.Settler, players.Loyal}},
	} {
		assert.Equal(t, c.want, players.TitlesOf(c.stats), "%+v", c.stats)
	}
}

func TestAStreakTitleCountsTheBestStreakNotTheCurrentOne(t *testing.T) {
	assert.Empty(t, players.TitlesOf(players.Stats{StreakCurrent: 30, StreakBest: 1}))
	assert.Equal(t, players.Titles{players.Loyal, players.Devoted}, players.TitlesOf(players.Stats{StreakCurrent: 0, StreakBest: 30}))
}

func TestWithoutLeavesOutTheHeldTitlesAndKeepsTheOrder(t *testing.T) {
	earned := players.Titles{players.Settler, players.Governor, players.Loyal}

	assert.Equal(t, players.Titles{players.Settler, players.Loyal}, earned.Without(players.Titles{players.Governor}))
	assert.Empty(t, earned.Without(earned))
	assert.Equal(t, players.Titles{players.Settler, players.Governor, players.Loyal}, earned, "the receiver is left as it was")
}

func TestSortedFollowsTheLadderAndDropsATitleItNoLongerHas(t *testing.T) {
	held := players.Titles{players.Devoted, "retired", players.Conqueror, players.Settler, players.Loyal}

	assert.Equal(t, players.Titles{players.Settler, players.Conqueror, players.Loyal, players.Devoted}, held.Sorted())
}
