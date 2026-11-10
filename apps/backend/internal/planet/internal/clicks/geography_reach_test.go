package clicks_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

const (
	centreTile = 1
	loneIsland = 20
	patchTiles = 19
)

func honeycomb(t *testing.T) *clicks.Geography {
	t.Helper()

	coords := patchCoords()
	require.Len(t, coords, patchTiles)

	geography, err := clicks.NewGeography(patchPositions(), touchingPairs(coords))
	require.NoError(t, err)

	return geography
}

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

func TestWithinIsACircleOfArcAroundThePoint(t *testing.T) {
	geography := equatorRow(t)

	assert.Equal(t, []uint32{1, 2}, geography.Within(clicks.Vec3{X: 1}, 0.15))
	assert.Equal(t, []uint32{1, 2, 3}, geography.Within(clicks.Vec3{X: 5}, 0.25), "any length of centre will do")
	assert.Empty(t, geography.Within(clicks.Vec3{}, 1))
}

func TestWithinReachesTilesNoEdgeLeadsTo(t *testing.T) {
	positions := make([]float32, 0, 6)
	for _, a := range []float64{0, 0.05} {
		positions = append(positions, float32(math.Cos(a)), float32(math.Sin(a)), 0)
	}
	across, err := clicks.NewGeography(positions, nil)
	require.NoError(t, err)

	assert.Equal(t, []uint32{1, 2}, across.Within(clicks.Vec3{X: 1}, 0.1),
		"two islands with water between them are both in the blast")
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
