package standings_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
)

func tallyOf(countries ...standings.Country) standings.Tally {
	var tally standings.Tally
	for position, country := range countries {
		tally = tally.WithTake(standings.Take{Country: country, Position: uint64(position)})
	}
	return tally
}

func TestTheFirstTakeMakesItsFlagTheMainOne(t *testing.T) {
	tally := tallyOf("fr")

	assert.Equal(t, standings.Tally{Main: "fr", Tiles: map[standings.Country]uint64{"fr": 1}, TakesBelow: 1}, tally)
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

	_ = before.WithTake(standings.Take{Country: "fr", Position: 1})

	assert.Equal(t, uint64(1), before.Tiles["fr"])
}

func TestATakeAtOrBeforeTheLastOneCountedIsCountedAlready(t *testing.T) {
	tally := tallyOf("fr", "de", "de")

	assert.Equal(t, tally, tally.WithTake(standings.Take{Country: "de", Position: 2}), "told again")
	assert.Equal(t, tally, tally.WithTake(standings.Take{Country: "fr", Position: 0}), "older still")
	assert.Equal(t, uint64(2), tally.WithTake(standings.Take{Country: "fr", Position: 3}).Tiles["fr"])
}
