package rounds_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
)

var day = rounds.Round{EndsAt: utc(16, 21, 0, 0)}

func TestTheCountryThatHeldTheMostGroundWinsTheRound(t *testing.T) {
	results := day.Results(map[rounds.Country]uint64{"de": 20, "fr": 30, "es": 10})

	assert.Equal(t, []rounds.Result{
		{Country: "fr", Rank: 1, Points: 25},
		{Country: "de", Rank: 2, Points: 18},
		{Country: "es", Rank: 3, Points: 15},
	}, results)
}

func TestCountriesThatHeldTheSameGroundShareTheRankAndItsPoints(t *testing.T) {
	results := day.Results(map[rounds.Country]uint64{"it": 20, "fr": 30, "de": 20, "es": 10})

	assert.Equal(t, []rounds.Result{
		{Country: "fr", Rank: 1, Points: 25},
		{Country: "de", Rank: 2, Points: 18},
		{Country: "it", Rank: 2, Points: 18},
		{Country: "es", Rank: 4, Points: 12},
	}, results)
}

func TestOnlyTheTopTenScore(t *testing.T) {
	held := map[rounds.Country]uint64{}
	for i := range 12 {
		held[rounds.Country(fmt.Sprintf("c%02d", i))] = uint64(100 - i)
	}

	results := day.Results(held)

	require.Len(t, results, 12)
	points := make([]uint32, 0, len(results))
	for _, result := range results {
		points = append(points, result.Points)
	}
	assert.Equal(t, []uint32{25, 18, 15, 12, 10, 8, 6, 4, 2, 1, 0, 0}, points)
}

func TestTheFinaleScoresThreeTimesAsMuch(t *testing.T) {
	finale := rounds.Round{EndsAt: utc(31, 23, 0, 0), Finale: true}

	results := finale.Results(map[rounds.Country]uint64{"fr": 30, "de": 20})

	assert.Equal(t, []rounds.Result{
		{Country: "fr", Rank: 1, Points: 75},
		{Country: "de", Rank: 2, Points: 54},
	}, results)
}

func TestACountryThatHeldNothingIsNotRanked(t *testing.T) {
	assert.Equal(t, []rounds.Result{{Country: "fr", Rank: 1, Points: 25}},
		day.Results(map[rounds.Country]uint64{"fr": 1, "de": 0}))
	assert.Empty(t, day.Results(map[rounds.Country]uint64{}))
}
