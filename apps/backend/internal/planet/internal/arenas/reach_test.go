package arenas_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// A row of land, 1 to 4, and two islands, 6 and 9. The water between 1 and 6 is one body, between 4 and 9 another.
var row = neighbourhood{1: {2}, 2: {1, 3}, 3: {2, 4}, 4: {3}, 6: {}, 9: {}}

type neighbourhood map[uint32][]uint32

func (n neighbourhood) Neighbours(id uint32) []uint32 { return n[id] }

type board map[uint32]string

func (b board) HoldsAny(country string, tiles []uint32) bool {
	for _, tile := range tiles {
		if b[tile] == country {
			return true
		}
	}
	return false
}

func reach(t *testing.T, owners board, open bool) arenas.Reach {
	t.Helper()
	waters, err := clicks.NewWaters(10, [][]uint32{{1, 6}, {4, 9}})
	require.NoError(t, err)
	clock := cptime.NewFixedClock(time.Date(2026, 11, 12, 20, 0, 0, 0, time.UTC))
	inPlay := arenas.NewInPlay(func() arenas.Rules { return arenas.Rules{} }, clock)
	if open {
		inPlay.Open(arenas.Opening{ID: "battle", OpenedAt: clock.Now(), EndsAt: clock.Now().Add(time.Hour)})
	}
	return arenas.NewReach(inPlay, row, waters, owners)
}

func TestInAnArenaAFlagTakesNextToItsLand(t *testing.T) {
	r := reach(t, board{1: "fr"}, true)

	require.NoError(t, r.Refuses("fr", 2))
	require.NoError(t, r.Refuses("fr", 1), "its own tile")
	require.ErrorIs(t, r.Refuses("fr", 3), arenas.ErrOutOfReach)
	require.ErrorIs(t, r.Refuses("de", 2), arenas.ErrOutOfReach, "a flag with no land reaches nothing")
}

func TestInAnArenaAFlagCrossesAWaterWhereItHoldsACoast(t *testing.T) {
	r := reach(t, board{1: "fr"}, true)

	require.NoError(t, r.Refuses("fr", 6), "across the water it holds a coast of")
	require.ErrorIs(t, r.Refuses("fr", 9), arenas.ErrOutOfReach, "no coast of that water")
}

func TestOutsideAnArenaEveryTileIsInReach(t *testing.T) {
	r := reach(t, board{1: "fr"}, false)

	require.NoError(t, r.Refuses("fr", 3))
	require.NoError(t, r.Refuses("de", 9))
	assert.Equal(t, []uint32{2, 4}, r.Around("fr", 3), "the map's own neighbours")
}

func TestASpreadReachesPastTheTileItTook(t *testing.T) {
	r := reach(t, board{1: "fr", 2: "fr"}, true)

	assert.Equal(t, []uint32{1, 3}, r.Around("fr", 2))
}

func TestASpreadFromAStruckTileTakesOnlyWhatTouchesOtherLand(t *testing.T) {
	r := reach(t, board{1: "fr", 2: "de"}, true)

	assert.Equal(t, []uint32{1}, r.Around("fr", 2), "3 touches only de's tile and fr's nothing")
}
