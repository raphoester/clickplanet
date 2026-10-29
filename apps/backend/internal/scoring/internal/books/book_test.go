package books_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
)

func TestABookHoldsTheTakesFromItsStartAndBeforeItsEndOnceClosed(t *testing.T) {
	open := books.NewBook("season-0", utc(8, 0))
	closed := open.ClosedAt(utc(31, 23), "")

	assert.False(t, open.Holds(utc(8, 0).Add(-time.Nanosecond)))
	assert.True(t, open.Holds(utc(8, 0)))
	assert.True(t, open.Holds(utc(31, 23).Add(time.Hour)), "an open book has no end")
	assert.True(t, closed.Holds(utc(31, 23).Add(-time.Nanosecond)), "a take told after the close still counts")
	assert.False(t, closed.Holds(utc(31, 23)))
}

func TestARoundHoldsTheTakesFromItsStartAndBeforeItsEndOnceScored(t *testing.T) {
	scored := day.ScoredAt(utc(16, 21), books.Snapshot{Tiles: 1})

	assert.False(t, day.Holds(utc(15, 21).Add(-time.Nanosecond)))
	assert.True(t, day.Holds(utc(15, 21)))
	assert.True(t, day.Holds(utc(17, 21)))
	assert.True(t, scored.Round.Holds(utc(16, 21).Add(-time.Nanosecond)))
	assert.False(t, scored.Round.Holds(utc(16, 21)))
}

func TestARoundScoresOneToTenTimes(t *testing.T) {
	for _, times := range []uint32{0, 11} {
		_, err := books.NewRound("battle", "season-0", times, utc(31, 21))
		require.ErrorIs(t, err, books.ErrInvalidTimes)
		require.ErrorIs(t, err, books.ErrRefused)
	}
	round, err := books.NewRound("battle", "season-0", 10, utc(31, 21))
	require.NoError(t, err)
	assert.Equal(t, books.Round{ID: "battle", Book: "season-0", Times: 10, StartsAt: utc(31, 21)}, round)
}

func TestAGrantGivesPointsToACountry(t *testing.T) {
	for _, grant := range []books.Grant{{Country: "", Points: 1}, {Country: "fr", Points: 0}} {
		_, err := books.NewGrant("grant", grant.Country, grant.Points, utc(9, 0))
		require.ErrorIs(t, err, books.ErrEmptyGrant)
		require.ErrorIs(t, err, books.ErrRefused)
	}
	grant, err := books.NewGrant("grant", "fr", 2, utc(9, 0).In(time.FixedZone("CEST", 2*3600)))
	require.NoError(t, err)
	assert.Equal(t, books.Grant{ID: "grant", Country: "fr", Points: 2, At: utc(9, 0)}, grant)
}

func TestAnIDIsOneToSixtyFourCharacters(t *testing.T) {
	for _, id := range []string{"", strings.Repeat("a", 65)} {
		_, err := books.ParseBookID(id)
		require.ErrorIs(t, err, books.ErrInvalidID)
		_, err = books.ParseRoundID(id)
		require.ErrorIs(t, err, books.ErrInvalidID)
		_, err = books.ParseGrantID(id)
		require.ErrorIs(t, err, books.ErrInvalidID)
	}
	id, err := books.ParseRoundID(strings.Repeat("a", 64))
	require.NoError(t, err)
	assert.Equal(t, books.RoundID(strings.Repeat("a", 64)), id)
}
