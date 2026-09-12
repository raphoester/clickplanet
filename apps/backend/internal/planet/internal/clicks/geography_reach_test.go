package clicks_test

import (
	"math"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

const (
	centreTile = 1  // the middle of the patch, by construction below
	loneIsland = 20 // in the map, in no edge
	patchTiles = 19 // 1 + 6 + 12: the centre and two full rings
)

// A hand-built patch of honeycomb — every tile within two steps of a centre — plus one tile with
// no edges at all. Built here rather than read off the shipped blob so that these tests say what
// Geography does; what the real map contains is geodesic_map's to prove.
func honeycomb(t *testing.T) *clicks.Geography {
	t.Helper()

	coords := patchCoords()
	require.Len(t, coords, patchTiles)

	geography, err := clicks.NewGeography(patchPositions(), touchingPairs(coords))
	require.NoError(t, err)

	return geography
}

// patchCoords lists the patch in axial hex coordinates, closest ring first, so the centre comes
// out as tile 1. A coordinate's tile id is its position here, plus one.
func patchCoords() [][2]int {
	var coords [][2]int

	for ring := range 3 {
		for q := -2; q <= 2; q++ {
			for r := -2; r <= 2; r++ {
				if hexDistance(q, r) == ring {
					coords = append(coords, [2]int{q, r})
				}
			}
		}
	}

	return coords
}

func touchingPairs(coords [][2]int) []clicks.Edge {
	ids := make(map[[2]int]uint32, len(coords))
	for i, coord := range coords {
		ids[coord] = uint32(i) + 1
	}

	var edges []clicks.Edge
	for _, from := range coords {
		for _, to := range coords {
			if hexDistance(from[0]-to[0], from[1]-to[1]) == 1 {
				edges = append(edges, clicks.Edge{From: ids[from], To: ids[to]})
			}
		}
	}

	return edges
}

// The lone island is the last tile and appears in no edge. These tests are about adjacency, so any
// unit vector will do for a position.
func patchPositions() []float32 {
	positions := make([]float32, loneIsland*3)
	for i := range loneIsland {
		positions[i*3] = 1
	}

	return positions
}

func hexDistance(q, r int) int {
	return (abs(q) + abs(r) + abs(q+r)) / 2
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func TestADiscGrowsOutwardsOneRingAtATime(t *testing.T) {
	geography := honeycomb(t)

	assert.Equal(t, []uint32{centreTile}, geography.Disc(centreTile, 0))
	assert.Len(t, geography.Disc(centreTile, 1), 7, "the centre and its six neighbours")
	assert.Len(t, geography.Disc(centreTile, 2), 19, "1 + 3k(k+1) at k=2")
}

func TestADiscStopsWhereTheLandDoes(t *testing.T) {
	geography := honeycomb(t)

	assert.Len(t, geography.Disc(centreTile, 9), patchTiles,
		"a radius past the edge of the land stops at the land")
	assert.Equal(t, []uint32{loneIsland}, geography.Disc(loneIsland, 1),
		"a lone island has nobody to spread to")
	assert.Empty(t, geography.Neighbours(loneIsland))
}

func TestADiscNeverRepeatsATileHoweverManyWaysItIsReached(t *testing.T) {
	geography := honeycomb(t)

	disc := geography.Disc(centreTile, 2)

	seen := map[uint32]bool{}
	for _, tile := range disc {
		require.False(t, seen[tile], "tile %d appears twice", tile)
		seen[tile] = true
	}
	assert.True(t, sortedAscending(disc))
}

func TestDiscRefusesATileThatIsNotOne(t *testing.T) {
	geography := honeycomb(t)

	assert.Empty(t, geography.Disc(0, 1))
	assert.Empty(t, geography.Disc(loneIsland+1, 1))
}

func TestDiscsAreSafeToTakeConcurrently(t *testing.T) {
	// The map is shared and immutable; the search scratch is neither.
	geography := honeycomb(t)
	want := geography.Disc(centreTile, 2)

	var wg sync.WaitGroup
	for range 64 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 50 {
				assert.Equal(t, want, geography.Disc(centreTile, 2))
			}
		}()
	}
	wg.Wait()
}

func TestPositionRefusesATileThatIsNotOne(t *testing.T) {
	geography := honeycomb(t)

	_, ok := geography.Position(0)
	assert.False(t, ok, "there is no tile 0")

	_, ok = geography.Position(loneIsland + 1)
	assert.False(t, ok, "one past the end is not a tile")

	_, ok = geography.Position(loneIsland)
	assert.True(t, ok, "the highest tile id is the tile count itself")
}

func equatorRow(t *testing.T) *clicks.Geography {
	t.Helper()

	positions := make([]float32, 0, 9)
	for _, a := range []float64{0, 0.1, 0.2} {
		positions = append(positions, float32(math.Cos(a)), float32(math.Sin(a)), 0)
	}

	geography, err := clicks.NewGeography(positions, []clicks.Edge{
		{From: 1, To: 2}, {From: 2, To: 1}, {From: 2, To: 3}, {From: 3, To: 2},
	})
	require.NoError(t, err)

	return geography
}

func TestSpacingIsTheMeanArcBetweenTouchingTiles(t *testing.T) {
	assert.InDelta(t, 0.1, equatorRow(t).Spacing(), 1e-6)
}

func TestNearestFindsTheClosestTileAndHowFarItIs(t *testing.T) {
	tile, arc := equatorRow(t).Nearest(clicks.Vec3{X: 3 * math.Cos(0.13), Y: 3 * math.Sin(0.13)})

	assert.Equal(t, uint32(2), tile)
	assert.InDelta(t, 0.03, arc, 1e-6)
}

func TestNearestRefusesAPointWithNoDirection(t *testing.T) {
	tile, _ := equatorRow(t).Nearest(clicks.Vec3{})

	assert.Zero(t, tile)
}

func sortedAscending(tiles []uint32) bool {
	for i := 1; i < len(tiles); i++ {
		if tiles[i-1] >= tiles[i] {
			return false
		}
	}
	return true
}
