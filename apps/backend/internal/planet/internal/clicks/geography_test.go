package clicks_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// The refusal tests call clicks.NewGeography themselves, because the error is what they assert on.
func threeTiles(t *testing.T, edges ...clicks.Edge) *clicks.Geography {
	t.Helper()

	geography, err := clicks.NewGeography(threeTilePositions(), edges)
	require.NoError(t, err)

	return geography
}

func threeTilePositions() []float32 {
	return make([]float32, 3*3)
}

func bothWays(a, b uint32) []clicks.Edge {
	return []clicks.Edge{{From: a, To: b}, {From: b, To: a}}
}

func TestNeighboursComeBackAscendingAndDeduplicated(t *testing.T) {
	// The walk finds a tile on a seam from both faces that share it, so repeats are expected
	// input, not a fault. Ascending order is a contract: the symmetry check binary-searches it.
	edges := append(bothWays(1, 3), bothWays(1, 2)...)
	edges = append(edges, bothWays(1, 3)...)

	geography := threeTiles(t, edges...)

	assert.Equal(t, []uint32{2, 3}, geography.Neighbours(1))
	assert.Equal(t, uint32(2), geography.Stats().Edges, "undirected, and counted once each")
}

func TestAnEdgeOnlyOneEndAgreesWithIsRefused(t *testing.T) {
	// A one-way street on the map: tile 1 would spread onto 2, and 2 would not spread back.
	_, err := clicks.NewGeography(threeTilePositions(), []clicks.Edge{{From: 1, To: 2}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not list")
}

func TestATileOutsideTheMapIsRefused(t *testing.T) {
	_, err := clicks.NewGeography(threeTilePositions(), bothWays(1, 4))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "outside 1..3")

	_, err = clicks.NewGeography(threeTilePositions(), bothWays(0, 1))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "outside 1..3")
}

func TestATileTouchingItselfIsRefused(t *testing.T) {
	_, err := clicks.NewGeography(threeTilePositions(), []clicks.Edge{{From: 2, To: 2}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "touches itself")
}

func TestMoreNeighboursThanATileCanHaveIsRefused(t *testing.T) {
	// Seven is the signature of a radius search, which is not how adjacency is found here.
	positions := make([]float32, 8*3)

	var edges []clicks.Edge
	for other := uint32(2); other <= 8; other++ {
		edges = append(edges, bothWays(1, other)...)
	}

	_, err := clicks.NewGeography(positions, edges)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "more than the 6")
}

func TestTheDegreeHistogramCountsEveryTileExactlyOnce(t *testing.T) {
	geography := threeTiles(t, bothWays(1, 2)...)

	stats := geography.Stats()
	assert.Equal(t, uint32(3), stats.Tiles)
	assert.Equal(t, uint32(1), stats.Degrees[0], "tile 3 touches nothing")
	assert.Equal(t, uint32(2), stats.Degrees[1], "tiles 1 and 2 touch each other")

	total := uint32(0)
	for _, count := range stats.Degrees {
		total += count
	}
	assert.Equal(t, stats.Tiles, total)
}

func TestAMapWithNoTilesIsRefused(t *testing.T) {
	_, err := clicks.NewGeography(nil, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no tiles")
}

func TestATileIsAllowedToTouchNothing(t *testing.T) {
	// 186 tiles on the real map are single-tile islands, so this is the normal case, not an edge.
	geography := threeTiles(t)

	assert.Empty(t, geography.Neighbours(1))
	assert.Equal(t, uint32(0), geography.Stats().Edges)
}
