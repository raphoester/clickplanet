package embedded_board_test

import (
	"encoding/binary"
	"encoding/json"
	"io/fs"
	"log/slog"
	"path"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/embedded_board"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpboards"
)

// What the generator cut (docs/provinces.md): a move in any number is another cut, which drops the locks.
var shippedProvinces = map[clicks.BoardName]struct {
	asset     string
	provinces int
	floor     uint32
	ofFloor   int
	largest   uint32
}{
	"earth-2": {asset: "provinces-286f4c3e.bin", provinces: 2097, floor: 200, ofFloor: 1090, largest: 93872},
	"siege-1": {asset: "provinces-8b268165.bin", provinces: 782, floor: 20, ofFloor: 516, largest: 2537},
}

func TestEveryTileOfABoardWithProvincesIsInOneOnItsLandmass(t *testing.T) {
	for name, want := range shippedProvinces {
		t.Run(string(name), func(t *testing.T) {
			ground, err := embedded_board.Shipped(name)
			require.NoError(t, err)
			provinces := ground.Provinces()
			assert.Equal(t, want.asset, provinces.Asset())
			assert.Equal(t, want.provinces, provinces.Count())

			sizes := map[clicks.ProvinceID]uint32{}
			for tile := uint32(1); tile <= ground.Tiles(); tile++ {
				province := provinces.ProvinceOf(tile)
				require.NotEqual(t, clicks.NoProvince, province, "tile %d", tile)
				require.Equal(t, ground.Borders().LandmassOf(tile), provinces.LandmassOf(province), "tile %d", tile)
				sizes[province]++
			}
			assert.Len(t, sizes, want.provinces)

			ofFloor, largest := 0, uint32(0)
			for province, size := range sizes {
				assert.Equal(t, provinces.Size(province), size)
				if size >= want.floor {
					ofFloor++
				}
				largest = max(largest, size)
			}
			assert.Equal(t, want.ofFloor, ofFloor)
			assert.Equal(t, want.largest, largest, "Antarctica, one province: no upper limit")
			assert.Equal(t, clicks.NoProvince, provinces.ProvinceOf(ground.Tiles()+1))
		})
	}
}

func TestEarth1HasNoProvinces(t *testing.T) {
	provinces := earth1.Provinces()

	assert.Equal(t, 0, provinces.Count())
	assert.Empty(t, provinces.Asset())
	assert.Equal(t, clicks.NoProvince, provinces.ProvinceOf(1))
}

func TestAProvincesFileOfOtherBordersIsRefused(t *testing.T) {
	for name, edit := range map[string]func(manifest *cpboards.Manifest, provinces []byte){
		"cut on other borders": func(_ *cpboards.Manifest, provinces []byte) {
			binary.LittleEndian.PutUint32(provinces[16:], binary.LittleEndian.Uint32(provinces[16:])+1)
		},
		"a tile in no province": func(_ *cpboards.Manifest, provinces []byte) {
			binary.LittleEndian.PutUint16(provinces[len(provinces)-2:], 0)
		},
		"a tile in a province of another landmass": func(_ *cpboards.Manifest, provinces []byte) {
			binary.LittleEndian.PutUint16(provinces[len(provinces)-2:], 1)
		},
		"another version": func(_ *cpboards.Manifest, provinces []byte) {
			binary.LittleEndian.PutUint32(provinces[4:], 2)
		},
	} {
		t.Run(name, func(t *testing.T) {
			registry, err := loadedRegistry(siegeWith(t, edit), func(string) bool { return true }, slog.New(slog.DiscardHandler))
			require.NoError(t, err)

			_, err = registry.Ground("siege-1")

			require.Error(t, err)
		})
	}
}

// siegeWith is siege-1 alone, its provinces file edited: the manifest is written again from what edit leaves of it.
func siegeWith(t *testing.T, edit func(manifest *cpboards.Manifest, provinces []byte)) fs.FS {
	t.Helper()
	board, err := cpboards.Named(cpboards.Files(), "siege-1")
	require.NoError(t, err)
	manifest := board.Manifest
	files := map[string][]byte{}
	for _, name := range []string{
		manifest.Files.Runs, manifest.Files.Borders, manifest.Files.Capitals, manifest.Files.Bridges,
		manifest.Files.Terrain, manifest.Files.Provinces,
	} {
		blob, err := board.File(name)
		require.NoError(t, err)
		files[name] = blob
	}
	edit(&manifest, files[manifest.Files.Provinces])
	blob, err := json.Marshal(manifest)
	require.NoError(t, err)
	edited := fstest.MapFS{path.Join("siege-1", board.Asset): {Data: blob}}
	for name, data := range files {
		edited[path.Join("siege-1", name)] = &fstest.MapFile{Data: data}
	}
	return edited
}
