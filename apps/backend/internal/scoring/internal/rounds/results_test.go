package rounds_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds"
)

var day = rounds.Round{EndsAt: utc(16, 21, 0, 0), Times: 1}

func atTheEnd(held map[rounds.Country]uint32) rounds.Snapshot {
	return rounds.Snapshot{Tiles: 100, Held: held}
}

func TestTheCountryThatHoldsTheMostGroundAtTheEndWinsTheRound(t *testing.T) {
	results := day.Results(atTheEnd(map[rounds.Country]uint32{"de": 20, "fr": 30, "es": 10}))

	assert.Equal(t, []rounds.Result{
		{Country: "fr", Rank: 1, Points: 25, Share: 0.3},
		{Country: "de", Rank: 2, Points: 18, Share: 0.2},
		{Country: "es", Rank: 3, Points: 15, Share: 0.1},
	}, results)
}

func TestCountriesThatHoldTheSameGroundShareTheRankAndItsPoints(t *testing.T) {
	results := day.Results(atTheEnd(map[rounds.Country]uint32{"it": 20, "fr": 30, "de": 20, "es": 10}))

	assert.Equal(t, []rounds.Result{
		{Country: "fr", Rank: 1, Points: 25, Share: 0.3},
		{Country: "de", Rank: 2, Points: 18, Share: 0.2},
		{Country: "it", Rank: 2, Points: 18, Share: 0.2},
		{Country: "es", Rank: 4, Points: 12, Share: 0.1},
	}, results)
}

func TestOnlyTheTopTenScore(t *testing.T) {
	held := map[rounds.Country]uint32{}
	for i := range 12 {
		held[rounds.Country(fmt.Sprintf("c%02d", i))] = uint32(100 - i) //nolint:gosec // 12 countries.
	}

	results := day.Results(rounds.Snapshot{Tiles: 2000, Held: held})

	require.Len(t, results, 12)
	points := make([]uint32, 0, len(results))
	for _, result := range results {
		points = append(points, result.Points)
	}
	assert.Equal(t, []uint32{25, 18, 15, 12, 10, 8, 6, 4, 2, 1, 0, 0}, points)
}

func TestARoundOfItsOwnScoresItsTimesADaysPoints(t *testing.T) {
	battle := rounds.Round{EndsAt: utc(31, 23, 0, 0), Own: true, StartsAt: utc(31, 21, 0, 0), Times: 3}

	results := battle.Results(atTheEnd(map[rounds.Country]uint32{"fr": 30, "de": 20}))

	assert.Equal(t, []rounds.Result{
		{Country: "fr", Rank: 1, Points: 75, Share: 0.3},
		{Country: "de", Rank: 2, Points: 54, Share: 0.2},
	}, results)
}

func TestACountryThatHoldsNothingAtTheEndIsNotRanked(t *testing.T) {
	assert.Equal(t, []rounds.Result{{Country: "fr", Rank: 1, Points: 25, Share: 0.01}},
		day.Results(atTheEnd(map[rounds.Country]uint32{"fr": 1, "de": 0})))
	assert.Empty(t, day.Results(atTheEnd(map[rounds.Country]uint32{})))
}
