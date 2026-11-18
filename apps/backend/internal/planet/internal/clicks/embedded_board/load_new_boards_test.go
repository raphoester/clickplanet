package embedded_board_test

import (
	"log/slog"
	"math"
	"runtime"
	"runtime/debug"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/embedded_board"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpboards"
)

// The boards PR 4 ships, which nothing plays yet: warped and rotated lattices whose tiles are numbered in canonical
// order (docs/boards.md). Their load, pinned as earth-1's is: a move in any number is another board.
var newBoards = map[clicks.BoardName]struct {
	asset   string
	tiles   uint32
	edges   uint32
	degrees [clicks.MaxDegree + 1]uint32
	// The size of each piece of land that touches, the largest first, and how many pieces.
	largest []int
	pieces  int
}{
	"earth-2": {
		asset:   "board-ea03c042.json",
		tiles:   1004682,
		edges:   2966323,
		degrees: [clicks.MaxDegree + 1]uint32{391, 1021, 2534, 10116, 16340, 14831, 959449},
		largest: []int{539916, 255116, 94085, 51996},
		pieces:  975,
	},
	"siege-1": {
		asset:   "board-95e08f3d.json",
		tiles:   27259,
		edges:   77006,
		degrees: [clicks.MaxDegree + 1]uint32{105, 159, 340, 897, 1373, 1320, 23065},
		largest: []int{14808, 3872, 3282, 2537},
		pieces:  173,
	},
}

func TestTheNewBoardsLoadEveryTileWhereThePagePutsIt(t *testing.T) {
	for name, want := range newBoards {
		t.Run(string(name), func(t *testing.T) {
			board, err := cpboards.Named(cpboards.Files(), string(name))
			require.NoError(t, err)
			assert.Equal(t, want.asset, board.Asset, "regenerating the board renumbers every tile")
			assert.Equal(t, "canonical", board.Manifest.Lattice.Order)

			ground, err := embedded_board.Shipped(name)
			require.NoError(t, err)
			geography := ground.Geography()
			stats := geography.Stats()
			assert.Equal(t, want.tiles, stats.Tiles)
			assert.Equal(t, want.edges, stats.Edges, "undirected edges")
			assert.Equal(t, want.degrees, stats.Degrees)

			hash := uint32(0x811c9dc5)
			for id := uint32(1); id <= want.tiles; id++ {
				position, ok := geography.Position(id)
				require.True(t, ok)
				for _, coordinate := range []float64{position.X, position.Y, position.Z} {
					hash = (hash ^ math.Float32bits(float32(coordinate))) * 0x01000193
				}
			}
			assert.Equal(t, board.Manifest.PositionsHash, hash, "FNV-1a over the float32 positions, as the page hashes them")

			sizes := landmassSizesOf(geography, int(want.tiles))
			assert.Len(t, sizes, want.pieces, "as many pieces as the bridges join")
			assert.Equal(t, want.largest, sizes[:4], "Afro-Eurasia, the Americas, Antarctica, Australia")
		})
	}
}

// What earth-2 costs the boot, logged for docs/boards.md: about 0.8 s and 300 MB allocated on a CI machine, and about
// 50 MB kept. Only a bound is asserted: a machine's speed is not the board's.
func TestEarth2LoadsInSecondsAndKeepsTensOfMegabytes(t *testing.T) {
	registry := embedded_board.New(cpboards.Files(), func(string) bool { return true }, slog.New(slog.DiscardHandler))
	require.NoError(t, registry.Load())
	debug.FreeOSMemory()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	started := time.Now()
	ground, err := registry.Ground("earth-2")
	took := time.Since(started)
	require.NoError(t, err)

	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	debug.FreeOSMemory()
	var kept runtime.MemStats
	runtime.ReadMemStats(&kept)
	t.Logf("earth-2: %d tiles in %s, %d MB allocated, %d MB kept", ground.Tiles(), took,
		(after.TotalAlloc-before.TotalAlloc)>>20, (kept.HeapAlloc-before.HeapAlloc)>>20)

	assert.Less(t, took, 20*time.Second, "the boot's deadline is a minute")
	assert.Less(t, kept.HeapAlloc-before.HeapAlloc, uint64(120<<20), "about 57 bytes a tile")
}
