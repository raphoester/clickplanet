package standings_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings"
)

func TestTakesAreEachTakeInTurn(t *testing.T) {
	takes := []standings.Take{
		{Country: "fr", Position: 0}, {Country: "de", Position: 1}, {Country: "de", Position: 1}, {Country: "de", Position: 2},
	}

	assert.Equal(t, standings.Tally{
		Main: "de", Tiles: map[standings.Country]uint64{"fr": 1, "de": 2}, TakesBelow: standings.TakeMark{Position: 2, Index: 1},
	}, standings.Tally{}.WithTakes(takes), "the take told twice counts once")
}
