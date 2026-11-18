package embedded_board_test

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"log/slog"
	"path"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/embedded_board"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpboards"
)

// loadedRegistry is a registry of the boards in files, their manifests read.
func loadedRegistry(files fs.FS, flags func(string) bool, logger *slog.Logger) (*embedded_board.Registry, error) {
	registry := embedded_board.New(files, fstest.MapFS{}, flags, logger)
	return registry, registry.Load() //nolint:wrapcheck // the registry names the board.
}

func registry(t *testing.T) *embedded_board.Registry {
	t.Helper()
	return registryLogging(t, &bytes.Buffer{})
}

func registryLogging(t *testing.T, logs *bytes.Buffer) *embedded_board.Registry {
	t.Helper()
	registry, err := loadedRegistry(cpboards.Files(), func(string) bool { return true },
		slog.New(slog.NewTextHandler(logs, nil)))
	require.NoError(t, err)
	return registry
}

// fixtureWith is the fixture alone, its files edited: the manifest is written again from what edit leaves of it.
func fixtureWith(t *testing.T, edit func(manifest *cpboards.Manifest, files map[string][]byte)) fs.FS {
	t.Helper()
	board := cpboards.FixtureBoard()
	files := map[string][]byte{}
	for _, name := range []string{
		board.Manifest.Files.Runs, board.Manifest.Files.Borders, board.Manifest.Files.Capitals,
		board.Manifest.Files.Bridges, board.Manifest.Files.Terrain,
	} {
		blob, err := board.File(name)
		require.NoError(t, err)
		files[name] = blob
	}
	manifest := board.Manifest
	edit(&manifest, files)
	blob, err := json.Marshal(manifest)
	require.NoError(t, err)
	edited := fstest.MapFS{path.Join(cpboards.Fixture, board.Asset): {Data: blob}}
	for name, data := range files {
		edited[path.Join(cpboards.Fixture, name)] = &fstest.MapFile{Data: data}
	}
	return edited
}

func TestASecondBoardIsLoadedTheFirstTimeItIsNeededAndKept(t *testing.T) {
	logs := &bytes.Buffer{}
	registry := registryLogging(t, logs)
	assert.Empty(t, logs.String(), "no ground is built at boot")

	fixture, err := registry.Ground(cpboards.Fixture)

	require.NoError(t, err)
	assert.Equal(t, clicks.BoardName(cpboards.Fixture), fixture.Name())
	assert.Equal(t, uint32(2998), fixture.Tiles())
	assert.Equal(t, 2998, int(fixture.Borders().Tiles()))
	assert.NotEmpty(t, fixture.Capitals().All())
	assert.Greater(t, fixture.Spacing(), 9*earth1.Spacing(), "about ten times as far apart as earth-1's")
	assert.Contains(t, logs.String(), "board=fixture-1")
	assert.NotContains(t, logs.String(), "board=earth-1", "earth-1 is not loaded until something needs it")

	again, err := registry.Ground(cpboards.Fixture)
	require.NoError(t, err)
	assert.Same(t, fixture, again, "built once")
	assert.Equal(t, 1, bytes.Count(logs.Bytes(), []byte("board loaded")))
}

func TestABoardThisProcessDoesNotCarryIsRefused(t *testing.T) {
	_, err := registry(t).Ground("moon-1")

	require.ErrorIs(t, err, eras.ErrNotCarried)
}

func TestEveryBoardCarriedNeedsPlanetsRowUnderItsHash(t *testing.T) {
	carried := registry(t).Carried()

	_, err := eras.BoardsOf(eras.Shipped, carried...)
	require.ErrorIs(t, err, eras.ErrUnkept, "the fixture has no row")
	assert.ErrorContains(t, err, "no row names board fixture-1")

	_, err = eras.BoardsOf([]eras.Board{eras.Earth1, eras.Fixture}, carried...)
	require.ErrorIs(t, err, eras.ErrUnkept, "earth-2 and siege-1 ship with their rows")

	moved := eras.Fixture
	moved.Hash = "deadbeef"
	_, err = eras.BoardsOf(append(slices.Clone(eras.Shipped), moved), carried...)
	require.ErrorIs(t, err, eras.ErrUnkept, "a row under another hash is another board")

	boards, err := eras.BoardsOf(append(slices.Clone(eras.Shipped), eras.Fixture), carried...)
	require.NoError(t, err)
	assert.Equal(t, clicks.BoardName(cpboards.Fixture), boards.Name(eras.Fixture.ID))
}

func TestAFileOfAnotherBoardIsRefused(t *testing.T) {
	for name, edit := range map[string]func(*cpboards.Manifest, map[string][]byte){
		"a manifest of another tile count": func(manifest *cpboards.Manifest, _ map[string][]byte) {
			manifest.Tiles--
		},
		"positions of another hash": func(manifest *cpboards.Manifest, _ map[string][]byte) {
			manifest.PositionsHash++
		},
		"borders of another board": func(manifest *cpboards.Manifest, files map[string][]byte) {
			borders, err := earth1Board().File(earth1Board().Manifest.Files.Borders)
			require.NoError(t, err)
			files[manifest.Files.Borders] = borders
		},
	} {
		t.Run(name, func(t *testing.T) {
			registry, err := loadedRegistry(fixtureWith(t, edit), func(string) bool { return true },
				slog.New(slog.DiscardHandler))
			require.NoError(t, err)

			_, err = registry.Ground(cpboards.Fixture)

			require.Error(t, err)
		})
	}
}

func TestABoardOfALatticeThisBuildCannotWalkIsRefusedAtBoot(t *testing.T) {
	for name, refused := range map[string]struct {
		edit func(*cpboards.Manifest)
		says string
	}{
		"an order no lattice has": {
			edit: func(manifest *cpboards.Manifest) { manifest.Lattice.Order = "spiral" },
			says: `tile order "spiral" is no lattice's`,
		},
		"corners that are not twelve": {
			edit: func(manifest *cpboards.Manifest) { manifest.Lattice.Corners = make([]float64, 33) },
			says: "corners of 33 numbers, not 36 finite ones",
		},
		"one col": {
			edit: func(manifest *cpboards.Manifest) { manifest.Lattice.Cols = 1 },
			says: "a lattice of 1 cols",
		},
	} {
		t.Run(name, func(t *testing.T) {
			files := fixtureWith(t, func(manifest *cpboards.Manifest, _ map[string][]byte) { refused.edit(manifest) })

			_, err := loadedRegistry(files, func(string) bool { return true }, slog.New(slog.DiscardHandler))

			require.ErrorContains(t, err, refused.says)
		})
	}
}

func TestAWarpedLatticeReadAsThreesIsRefused(t *testing.T) {
	files := fixtureWith(t, func(manifest *cpboards.Manifest, _ map[string][]byte) {
		manifest.Lattice.Warp = &cpboards.Warp{A: -0.09, B: -0.13, C: 0.26}
	})
	registry, err := loadedRegistry(files, func(string) bool { return true }, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	_, err = registry.Ground(cpboards.Fixture)

	require.ErrorContains(t, err, "they are not the page's", "the warp moves every position, so the hash")
}

func earth1Board() cpboards.Board {
	board, err := cpboards.Named(cpboards.Files(), "earth-1")
	if err != nil {
		panic(err)
	}
	return board
}
