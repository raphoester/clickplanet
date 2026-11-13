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
	assert.Equal(t, []uint32{2, 4}, r.Around("fr", 3, 1), "the map's own neighbours")
}

func TestASpreadReachesPastTheTileItTook(t *testing.T) {
	r := reach(t, board{1: "fr", 2: "fr"}, true)

	assert.Equal(t, []uint32{1, 3}, r.Around("fr", 2, 1))
}

func TestASpreadFromAStruckTileTakesOnlyWhatTouchesOtherLand(t *testing.T) {
	r := reach(t, board{1: "fr", 2: "de"}, true)

	assert.Equal(t, []uint32{1}, r.Around("fr", 2, 1), "3 touches only de's tile and fr's nothing")
}

func TestAWideSpreadFromATileTakenTakesItsWholeDisc(t *testing.T) {
	r := reach(t, board{1: "fr", 2: "fr"}, true)

	assert.Equal(t, []uint32{1, 3, 4}, r.Around("fr", 2, 2), "4 is out of reach, and the spread takes 3 on the way")
}

func TestAWideSpreadFromAStruckTileTakesOnlyTheTilesOfItsDiscInReach(t *testing.T) {
	r := reach(t, board{1: "fr", 3: "de"}, true)

	assert.Equal(t, []uint32{2, 1}, r.Around("fr", 3, 2), "4 touches only de's tile")
}

func TestWithoutTheReachASpreadTakesItsWholeDisc(t *testing.T) {
	r := reach(t, board{}, false)

	assert.Equal(t, []uint32{2, 4, 1}, r.Around("fr", 3, 2))
}

type hexGrid struct{}

func (hexGrid) Neighbours(id uint32) []uint32 {
	q, r := int(id-1)%9, int(id-1)/9
	var neighbours []uint32
	for _, step := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, -1}, {-1, 1}} {
		nq, nr := q+step[0], r+step[1]
		if nq >= 0 && nq < 9 && nr >= 0 && nr < 9 {
			neighbours = append(neighbours, uint32(1+nq+9*nr))
		}
	}
	return neighbours
}

func TestADiscIsEveryTileWithinItsRingsNearestFirst(t *testing.T) {
	const centre = 1 + 4 + 9*4

	for rings, size := range map[int]int{0: 0, 1: 6, 2: 18, 3: 36} {
		disc := clicks.Disc(hexGrid{}, centre, rings)

		assert.Len(t, disc, size, "%d rings", rings)
		assert.NotContains(t, disc, uint32(centre))
		if rings > 0 {
			assert.ElementsMatch(t, hexGrid{}.Neighbours(centre), disc[:6], "the first ring first")
		}
	}
}

func TestADiscStopsAtTheEdgeOfTheLand(t *testing.T) {
	assert.Equal(t, []uint32{2, 3, 4}, clicks.Disc(landRow, 1, 5))
	assert.Empty(t, clicks.Disc(landRow, 6, 2), "an island")
}

func TestASpreadNeverHandsBackTheMapsOwnTable(t *testing.T) {
	r := reach(t, board{}, false)

	r.Around("fr", 3, 1)[0] = 0

	assert.Equal(t, []uint32{2, 4}, landRow[3])
}
