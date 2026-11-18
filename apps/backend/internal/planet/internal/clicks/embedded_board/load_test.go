package embedded_board_test

import (
	"math"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mapdata "github.com/raphoester/clickplanet.lol-backend/generated/map"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/embedded_board"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpboards"
)

const tiles = 260503

var (
	earth1    = mustLoad()
	geography = earth1.Geography()
)

func mustLoad() *clicks.Ground {
	ground, err := embedded_board.Shipped("earth-1")
	if err != nil {
		panic(err)
	}
	return ground
}

func shipped(t *testing.T) (cpboards.Manifest, string) {
	t.Helper()
	board, err := cpboards.Named(mapdata.FS(), "earth-1")
	require.NoError(t, err)
	return board.Manifest, board.Asset
}

func TestTheShippedBoardIsEarth1(t *testing.T) {
	boards, err := cpboards.Boards(mapdata.FS())
	require.NoError(t, err)
	require.Len(t, boards, 1, "earth-1 alone until PR 4 ships earth-2 and siege-1")
	assert.Equal(t, "earth-1", boards[0].Manifest.Name)

	manifest, asset := shipped(t)
	assert.Equal(t, "board-725d8920.json", asset,
		"regenerating the board renumbers every tile and moves every player's territory")
	assert.Equal(t, "runs-ac17bdab.bin", manifest.Files.Runs)
	assert.Equal(t, 301, manifest.Lattice.Cols)
	assert.Equal(t, uint32(tiles), manifest.Tiles)
	assert.Equal(t, uint32(tiles), geography.Stats().Tiles)
}

func TestEachBoardCarriedIsNamedByItsDirectoryAndItsPositionsHash(t *testing.T) {
	assert.Equal(t, []eras.Carried{eras.Earth1Carried, eras.FixtureCarried}, registry(t).Carried(),
		"earth-1 as the migration that made boards keeps it, and the test binary's fixture")
}

func TestEveryPositionIsThePagesToTheBit(t *testing.T) {
	manifest, _ := shipped(t)

	hash := uint32(0x811c9dc5)
	for id := uint32(1); id <= tiles; id++ {
		position, ok := geography.Position(id)
		require.True(t, ok)
		for _, coordinate := range []float64{position.X, position.Y, position.Z} {
			hash = (hash ^ math.Float32bits(float32(coordinate))) * 0x01000193
		}
	}

	assert.Equal(t, manifest.PositionsHash, hash, "FNV-1a over the float32 positions, as the page and the backoffice hash them")
}

func TestEveryTileHasAPlausibleNumberOfNeighbours(t *testing.T) {
	assert.Equal(t, [clicks.MaxDegree + 1]uint32{
		0: 250,
		1: 527,
		2: 1154,
		3: 4072,
		4: 6368,
		5: 6471,
		6: 241661,
	}, geography.Stats().Degrees)
}

func TestTheEdgeCountAndAverageDegree(t *testing.T) {
	assert.Equal(t, uint32(761422), geography.Stats().Edges, "undirected edges")

	directed := 0
	for id := uint32(1); id <= tiles; id++ {
		directed += len(geography.Neighbours(id))
	}
	assert.Equal(t, 1522844, directed, "each undirected edge is listed from both ends")
	assert.InDelta(t, 5.846, float64(directed)/tiles, 0.001, "average degree")
}

func TestTileIDsAreOneBasedOverTheRuns(t *testing.T) {
	_, ok := geography.Position(0)
	assert.False(t, ok, "there is no tile 0")

	_, ok = geography.Position(tiles)
	assert.True(t, ok, "the highest tile id is maxIndex itself, not maxIndex-1")

	_, ok = geography.Position(tiles + 1)
	assert.False(t, ok, "one past the end is not a tile")

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

func TestAdjacencyAloneFindsTheContinents(t *testing.T) {
	sizes := landmassSizes(geography)

	assert.Len(t, sizes, 505, "connected landmasses")
	assert.Equal(t, []int{142792, 67352, 22871, 12402}, sizes[:4],
		"Afro-Eurasia, the Americas, Antarctica, Australia")
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

func TestASpreadOnTheShippedMapTakes7Or19Or37TilesAwayFromTheCoasts(t *testing.T) {
	full := map[int]int{1: 6, 2: 18, 3: 36}
	fullDiscs := map[int]int{}
	for tile := uint32(1); tile <= tiles; tile += 997 {
		for rings, size := range full {
			disc := clicks.Disc(geography, tile, rings)
			require.LessOrEqual(t, len(disc), size, "tile %d, %d rings", tile, rings)
			if len(disc) == size {
				fullDiscs[rings]++
			}
		}
	}

	for rings := range full {
		assert.Greater(t, fullDiscs[rings], 100, "%d rings: most land tiles are inland", rings)
	}
}
