package standings_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
)

func tallyOf(countries ...standings.Country) standings.Tally {
	var tally standings.Tally
	for _, country := range countries {
		tally = tally.WithTake(country)
	}
	return tally
}

func TestTheFirstTakeMakesItsFlagTheMainOne(t *testing.T) {
	tally := tallyOf("fr")

	assert.Equal(t, standings.Tally{Main: "fr", Tiles: map[standings.Country]uint64{"fr": 1}}, tally)
}

func TestATallyWithNoTakeIsEmpty(t *testing.T) {
	assert.True(t, standings.Tally{}.Empty())
	assert.False(t, tallyOf("fr").Empty())
}

func TestTheMainFlagIsTheOneWithTheMostTiles(t *testing.T) {
	assert.Equal(t, standings.Country("de"), tallyOf("fr", "de", "de").Main)
	assert.Equal(t, standings.Country("fr"), tallyOf("fr", "fr", "de", "it", "de").Main)
}

func TestATieKeepsTheFlagThatGotThereFirst(t *testing.T) {
	assert.Equal(t, standings.Country("fr"), tallyOf("fr", "de").Main)
	assert.Equal(t, standings.Country("de"), tallyOf("fr", "de", "de", "fr").Main)
}

func TestATakeLeavesTheTallyItWasAddedToAsItWas(t *testing.T) {
	before := tallyOf("fr")

	_ = before.WithTake("fr")

	assert.Equal(t, uint64(1), before.Tiles["fr"])
}

func TestTheLineIsTheMainFlagAndItsTilesOnly(t *testing.T) {
	account := standings.AccountID{15: 1}

	line := tallyOf("fr", "de", "de", "it").LineOf(account)

	assert.Equal(t, standings.Line{Account: account, Country: "de", Tiles: 2}, line)
}

func TestMoreTilesComeFirstThenTheLowerAccount(t *testing.T) {
	low := standings.Line{Account: standings.AccountID{15: 1}, Tiles: 2}
	high := standings.Line{Account: standings.AccountID{15: 2}, Tiles: 2}
	best := standings.Line{Account: standings.AccountID{15: 3}, Tiles: 3}

	assert.True(t, best.Above(low))
	assert.True(t, low.Above(high))
	assert.False(t, high.Above(low))
	assert.False(t, low.Above(low))
}

func TestTheStartComesBeforeEveryLineAndALineBeforeTheOnesBelowIt(t *testing.T) {
	first := standings.Line{Account: standings.AccountID{15: 1}, Tiles: 5}
	next := standings.Line{Account: standings.AccountID{15: 2}, Tiles: 5}

	assert.True(t, standings.Start.Before(first))
	assert.True(t, first.Cursor().Before(next))
	assert.False(t, first.Cursor().Before(first))
	assert.False(t, next.Cursor().Before(first))
}
