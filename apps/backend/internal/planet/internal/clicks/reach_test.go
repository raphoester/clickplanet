package clicks_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// landRow is a row of land, 1 to 4, and two islands, 6 and 9. A bridge leads from 1 to 6, and none to 9.
var landRow = neighbourhood{1: {2}, 2: {1, 3}, 3: {2, 4}, 4: {3}, 6: {}, 9: {}}

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

func reach(t *testing.T, owners board, on bool) clicks.Reach {
	t.Helper()
	bridges, err := clicks.NewBridges(10, [][2]uint32{{1, 6}})
	require.NoError(t, err)
	return clicks.NewReach(func() clicks.Reaching { return clicks.Reaching(on) }, landRow, bridges, owners)
}

func TestWithTheReachAFlagTakesNextToItsLand(t *testing.T) {
	r := reach(t, board{1: "fr"}, true)

	require.NoError(t, r.Refuses("fr", 2))
	require.NoError(t, r.Refuses("fr", 1), "its own tile")
	require.ErrorIs(t, r.Refuses("fr", 3), clicks.ErrOutOfReach)
	require.ErrorIs(t, r.Refuses("de", 2), clicks.ErrOutOfReach, "a flag with no land reaches nothing")
}

func TestWithTheReachAFlagCrossesABridgeFromItsLand(t *testing.T) {
	r := reach(t, board{1: "fr"}, true)

	require.NoError(t, r.Refuses("fr", 6), "the far end of a bridge from its land")
	require.ErrorIs(t, r.Refuses("fr", 9), clicks.ErrOutOfReach, "an island no bridge from its land leads to")
	require.ErrorIs(t, r.Refuses("de", 6), clicks.ErrOutOfReach, "a bridge from land it does not hold")
}

func TestWithTheReachABridgeLeadsBothWays(t *testing.T) {
	r := reach(t, board{6: "fr"}, true)

	require.NoError(t, r.Refuses("fr", 1))
}

func TestWithoutTheReachEveryTileIsInReach(t *testing.T) {
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
