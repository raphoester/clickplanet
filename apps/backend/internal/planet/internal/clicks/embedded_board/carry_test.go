package embedded_board_test

import (
	"encoding/binary"
	"io/fs"
	"log/slog"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mapdata "github.com/raphoester/clickplanet.lol-backend/generated/map"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/embedded_board"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpboards"
)

func registryWith(t *testing.T, carries fs.FS) *embedded_board.Registry {
	t.Helper()
	registry := embedded_board.New(cpboards.Files(), carries, func(string) bool { return true }, slog.New(slog.DiscardHandler))
	require.NoError(t, registry.Load())
	return registry
}

func TestTheCarryOverFromEarth1ToEarth2GivesNearlyEveryNewTileAParent(t *testing.T) {
	carry, err := registryWith(t, mapdata.Carries()).Carryover("earth-1", "earth-2")
	require.NoError(t, err)

	assert.Equal(t, uint32(1_004_682), carry.Tiles())
	assert.Equal(t, uint32(260_503), carry.OldTiles())
	carried, most, childless := 0, 0, 0
	for child := uint32(1); child <= carry.Tiles(); child++ {
		if carry.Parent(child) != 0 {
			carried++
		}
	}
	for parent := uint32(1); parent <= carry.OldTiles(); parent++ {
		children := len(carry.Children(parent))
		most = max(most, children)
		if children == 0 {
			childless++
		}
	}
	assert.Equal(t, 997_267, carried, "7,415 new tiles are new coasts and islands")
	assert.Equal(t, 7, most)
	assert.Equal(t, 66, childless, "66 old coast tiles earth-2 draws as sea")
}

func TestNoCarryOverLeadsBackNorToABoardNotCarried(t *testing.T) {
	registry := registryWith(t, mapdata.Carries())

	_, err := registry.Carryover("earth-2", "earth-1")
	require.ErrorIs(t, err, embedded_board.ErrNoCarryover)
	_, err = registry.Carryover("earth-1", "earth-3")
	require.ErrorIs(t, err, embedded_board.ErrNoCarryover)
}

func TestACarryOverOfBoardsRegeneratedSinceIsRefused(t *testing.T) {
	names, err := fs.Glob(mapdata.Carries(), "carry-earth-1-earth-2-*.bin")
	require.NoError(t, err)
	require.Len(t, names, 1)
	blob, err := fs.ReadFile(mapdata.Carries(), names[0])
	require.NoError(t, err)

	for at, what := range map[int]string{8: "earth-1's positions", 12: "earth-2's positions", 20: "earth-2's tiles"} {
		stale := append([]byte(nil), blob...)
		binary.LittleEndian.PutUint32(stale[at:], binary.LittleEndian.Uint32(stale[at:])+1)
		_, err := registryWith(t, fstest.MapFS{names[0]: {Data: stale}}).Carryover("earth-1", "earth-2")
		require.ErrorIs(t, err, clicks.ErrCarryover, what)
	}

	_, err = registryWith(t, fstest.MapFS{names[0]: {Data: blob[:len(blob)-1]}}).Carryover("earth-1", "earth-2")
	require.ErrorIs(t, err, clicks.ErrCarryover, "a file cut short")
}
