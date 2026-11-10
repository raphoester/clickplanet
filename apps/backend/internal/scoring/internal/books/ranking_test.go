package books_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
)

var heldAndGainDay = books.Round{ID: "season-1/day-10", Book: "season-1", Times: 1, Ranking: books.HeldAndGainRanking{}, StartsAt: utc(15, 21)}

func TestADayRankedOnTheGroundHeldAndGainedOrdersTheSumOfBothRanks(t *testing.T) {
	results := heldAndGainDay.Results(
		books.Start{"ie": 50, "br": 30, "fr": 15, "lv": 5},
		atTheEnd(map[books.Country]uint32{"ie": 50, "br": 28, "fr": 10, "lv": 12}),
	)

	assert.Equal(t, []books.Result{
		{Country: "ie", Rank: 1, Points: 25, Share: 0.5},
		{Country: "lv", Rank: 2, Points: 18, Share: 0.12},
		{Country: "br", Rank: 3, Points: 15, Share: 0.28},
		{Country: "fr", Rank: 4, Points: 12, Share: 0.1},
	}, results)
}

func TestTheGainsRankBreaksATieOfTheSum(t *testing.T) {
	results := heldAndGainDay.Results(books.Start{"ie": 50, "lv": 10}, atTheEnd(map[books.Country]uint32{"ie": 50, "lv": 20}))

	assert.Equal(t, []books.Result{
		{Country: "lv", Rank: 1, Points: 25, Share: 0.2},
		{Country: "ie", Rank: 2, Points: 18, Share: 0.5},
	}, results)
}

func TestCountriesEqualOnBothRanksShareThePlaceAndItsPoints(t *testing.T) {
	results := heldAndGainDay.Results(books.Start{"fr": 10, "de": 10, "es": 5}, atTheEnd(map[books.Country]uint32{"fr": 12, "de": 12, "es": 5}))

	assert.Equal(t, []books.Result{
		{Country: "de", Rank: 1, Points: 25, Share: 0.12},
		{Country: "fr", Rank: 1, Points: 25, Share: 0.12},
		{Country: "es", Rank: 3, Points: 15, Share: 0.05},
	}, results)
}

func TestADayWhenNobodyGainedRanksTheGroundHeld(t *testing.T) {
	results := heldAndGainDay.Results(books.Start{"fr": 50, "de": 10}, atTheEnd(map[books.Country]uint32{"fr": 50, "de": 10}))

	assert.Equal(t, []books.Result{
		{Country: "fr", Rank: 1, Points: 25, Share: 0.5},
		{Country: "de", Rank: 2, Points: 18, Share: 0.1},
	}, results)
}

func TestARankingIsFoundByItsName(t *testing.T) {
	for _, ranking := range []books.Ranking{books.GainRanking{}, books.HeldAndGainRanking{}} {
		found, err := books.RankingNamed(ranking.Name())
		require.NoError(t, err)
		assert.Equal(t, ranking, found)
	}
	_, err := books.RankingNamed("average")
	require.ErrorIs(t, err, books.ErrUnknownRanking)
}
