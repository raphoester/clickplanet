package books_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
)

func utc(day, hour int) time.Time {
	return time.Date(2026, 10, day, hour, 0, 0, 0, time.UTC)
}

var day = books.Round{ID: "season-0/day-9", Book: "season-0", Times: 1, StartsAt: utc(15, 21)}

func atTheEnd(held map[books.Country]uint32) books.Snapshot {
	return books.Snapshot{Tiles: 100, Held: held}
}

func TestTheCountryThatHoldsTheMostGroundAtTheEndWinsTheRound(t *testing.T) {
	results := day.Results(atTheEnd(map[books.Country]uint32{"de": 20, "fr": 30, "es": 10}))

	assert.Equal(t, []books.Result{
		{Country: "fr", Rank: 1, Points: 25, Share: 0.3},
		{Country: "de", Rank: 2, Points: 18, Share: 0.2},
		{Country: "es", Rank: 3, Points: 15, Share: 0.1},
	}, results)
}

func TestCountriesThatHoldTheSameGroundShareTheRankAndItsPoints(t *testing.T) {
	results := day.Results(atTheEnd(map[books.Country]uint32{"it": 20, "fr": 30, "de": 20, "es": 10}))

	assert.Equal(t, []books.Result{
		{Country: "fr", Rank: 1, Points: 25, Share: 0.3},
		{Country: "de", Rank: 2, Points: 18, Share: 0.2},
		{Country: "it", Rank: 2, Points: 18, Share: 0.2},
		{Country: "es", Rank: 4, Points: 12, Share: 0.1},
	}, results)
}

func TestOnlyTheTopTenScore(t *testing.T) {
	held := map[books.Country]uint32{}
	for i := range 12 {
		held[books.Country(fmt.Sprintf("c%02d", i))] = uint32(100 - i) //nolint:gosec // 12 countries.
	}

	results := day.Results(books.Snapshot{Tiles: 2000, Held: held})

	require.Len(t, results, 12)
	points := make([]uint32, 0, len(results))
	for _, result := range results {
		points = append(points, result.Points)
	}
	assert.Equal(t, []uint32{25, 18, 15, 12, 10, 8, 6, 4, 2, 1, 0, 0}, points)
}

func TestARoundScoresItsTimesThePoints(t *testing.T) {
	battle := books.Round{ID: "0192f5a0-7c00-7000-8000-000000000031", Book: "season-0", StartsAt: utc(31, 21), Times: 3}

	results := battle.Results(atTheEnd(map[books.Country]uint32{"fr": 30, "de": 20}))

	assert.Equal(t, []books.Result{
		{Country: "fr", Rank: 1, Points: 75, Share: 0.3},
		{Country: "de", Rank: 2, Points: 54, Share: 0.2},
	}, results)
}

func TestACountryThatHoldsNothingAtTheEndIsNotRanked(t *testing.T) {
	assert.Equal(t, []books.Result{{Country: "fr", Rank: 1, Points: 25, Share: 0.01}},
		day.Results(atTheEnd(map[books.Country]uint32{"fr": 1, "de": 0})))
	assert.Empty(t, day.Results(atTheEnd(map[books.Country]uint32{})))
}

func TestARoundScoredKeepsItsEndAndItsResults(t *testing.T) {
	scored := day.ScoredAt(utc(16, 21), atTheEnd(map[books.Country]uint32{"fr": 30}))

	assert.Equal(t, books.Scored{
		Round:   books.Round{ID: day.ID, Book: day.Book, Times: 1, StartsAt: day.StartsAt, Scored: true, EndsAt: utc(16, 21)},
		Results: []books.Result{{Country: "fr", Rank: 1, Points: 25, Share: 0.3}},
	}, scored)
}
