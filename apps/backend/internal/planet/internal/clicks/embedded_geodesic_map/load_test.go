package embedded_geodesic_map_test

import (
	"math"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mapdata "github.com/raphoester/clickplanet.lol-backend/generated/map"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/embedded_geodesic_map"
)

// gameMap.maxIndex in cmd/api/example.yaml. Written out, not read from the map, so that a blob
// swapped underneath this suite fails it.
const tiles = 262119

var geography = mustLoad()

func mustLoad() *clicks.Geography {
	geography, err := embedded_geodesic_map.New(tiles, nil).LoadGeography()
	if err != nil {
		panic(err)
	}
	return geography
}

func TestTheShippedBlobIsADetail300Honeycomb(t *testing.T) {
	_, asset, err := mapdata.Coordinates()
	require.NoError(t, err)

	assert.Equal(t, "coordinates-9998a414.bin", asset,
		"regenerating the blob renumbers every tile and moves every player's territory")
	assert.Equal(t, uint32(tiles), geography.Stats().Tiles)
	assert.Equal(t, 300, embedded_geodesic_map.Detail)
	assert.Equal(t, 301, embedded_geodesic_map.Cols)
}

func TestEveryTileHasAPlausibleNumberOfNeighbours(t *testing.T) {
	assert.Equal(t, [clicks.MaxDegree + 1]uint32{
		0: 225,    // single-tile islands, with nobody to spread to
		1: 516,    //
		2: 1111,   //
		3: 3796,   // coastlines
		4: 5664,   //
		5: 5048,   // includes whichever of the 12 icosahedron corners are land
		6: 245759, // inland
	}, geography.Stats().Degrees)
}

func TestTheEdgeCountAndAverageDegree(t *testing.T) {
	assert.Equal(t, uint32(768288), geography.Stats().Edges, "undirected edges")

	directed := 0
	for id := uint32(1); id <= tiles; id++ {
		directed += len(geography.Neighbours(id))
	}
	assert.Equal(t, 1536576, directed, "each undirected edge is listed from both ends")
	assert.InDelta(t, 5.862, float64(directed)/tiles, 0.001, "average degree")
}

func TestTileIDsAreOneBasedOverTheBlob(t *testing.T) {
	// Shift this by one and every neighbourhood is one tile off, symmetrically, with a degree
	// histogram that still looks right.
	_, ok := geography.Position(0)
	assert.False(t, ok, "there is no tile 0")

	_, ok = geography.Position(tiles)
	assert.True(t, ok, "the highest tile id is maxIndex itself, not maxIndex-1")

	_, ok = geography.Position(tiles + 1)
	assert.False(t, ok, "one past the end is not a tile")

	// Tile 1 is the blob's first entry: the first three floats of coordinates-9998a414.bin.
	first, ok := geography.Position(1)
	require.True(t, ok)
	assert.InDelta(t, -0.5943395495414734, first.X, 1e-9)
	assert.InDelta(t, 0.8018954992294312, first.Y, 1e-9)
	assert.InDelta(t, 0.06102520972490311, first.Z, 1e-9)
}

func TestEveryPositionIsOnTheUnitSphere(t *testing.T) {
	for id := uint32(1); id <= tiles; id++ {
		position, ok := geography.Position(id)
		require.True(t, ok)

		length := math.Sqrt(position.X*position.X + position.Y*position.Y + position.Z*position.Z)
		require.InDelta(t, 1.0, length, 1e-6, "tile %d", id)
	}
}

func TestADiscInlandIsTheHoneycombSeries(t *testing.T) {
	// deepInlandTile is far enough from any coast that its radius-5 disc is all land. Written
	// down rather than searched for, so a map that lost its inland runs fails here instead of
	// picking another tile and passing.
	const deepInlandTile = 14

	for radius, expected := range map[int]int{1: 7, 2: 19, 3: 37, 4: 61, 5: 91} {
		assert.Len(t, geography.Disc(deepInlandTile, radius), expected, "radius %d", radius)
	}
}

func TestNoDiscIsBiggerThanTheHoneycombSeries(t *testing.T) {
	const radius = 3
	ceiling := 1 + 3*radius*(radius+1)

	for id := uint32(1); id <= tiles; id += 97 {
		require.LessOrEqual(t, len(geography.Disc(id, radius)), ceiling, "tile %d", id)
	}
}

func TestAdjacencyAloneFindsTheContinents(t *testing.T) {
	// 20 faces left unstitched would show up here as far more pieces than there is land.
	sizes := landmassSizes(geography)

	assert.Len(t, sizes, 477, "connected landmasses")
	assert.Equal(t, []int{143574, 68214, 22871, 12405}, sizes[:4],
		"Afro-Eurasia, the Americas, Antarctica, Australia")
}

func TestLoadRefusesABlobThatDisagreesWithTheConfiguredTileCount(t *testing.T) {
	_, err := embedded_geodesic_map.New(tiles-1, nil).LoadGeography()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "updated apart")
}

func landmassSizes(g *clicks.Geography) []int {
	seen := make([]bool, tiles+1)
	var sizes []int

	for id := uint32(1); id <= tiles; id++ {
		if seen[id] {
			continue
		}

		size := 0
		seen[id] = true
		stack := []uint32{id}

		for len(stack) > 0 {
			tile := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			size++

			for _, neighbour := range g.Neighbours(tile) {
				if !seen[neighbour] {
					seen[neighbour] = true
					stack = append(stack, neighbour)
				}
			}
		}

		sizes = append(sizes, size)
	}

	slices.SortFunc(sizes, func(a, b int) int { return b - a })
	return sizes
}
