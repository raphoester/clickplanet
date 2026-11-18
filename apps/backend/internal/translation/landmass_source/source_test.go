package landmass_source_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/translation"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation/landmass_source"
)

type gazetteer map[string]map[string]string

func (g gazetteer) NamesOf(english string) map[string]string {
	return g[english]
}

func TestALandmassCarriesNaturalEarthsNamesAndKeepsItsNumber(t *testing.T) {
	path := filepath.Join(t.TempDir(), "landmassNames.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"borders": "borders-1.bin", "names": {"20": "Ionian Islands 2", "13": "Split-Dalmatia", "9": "Atlantis"}}`), 0o600))
	known := gazetteer{
		"Ionian Islands": {"fr": "Îles Ioniennes", "de": "Ionische Inseln"},
		"Split-Dalmatia": {"fr": "Split-Dalmatie"},
	}

	lines, order, err := landmass_source.Read(path, known)

	require.NoError(t, err)
	assert.Equal(t, []translation.Key{"20", "13", "9"}, order)
	assert.Equal(t, []translation.Line{
		{Key: "20", English: "Ionian Islands 2", Context: `A place on the world map: an island, a region or a province. " 2" tells it from another place named "Ionian Islands": keep it. Natural Earth names it: de "Ionische Inseln", fr "Îles Ioniennes".`},
		{Key: "13", English: "Split-Dalmatia", Context: `A place on the world map: an island, a region or a province. Natural Earth names it: fr "Split-Dalmatie".`},
		{Key: "9", English: "Atlantis", Context: "A place on the world map: an island, a region or a province."},
	}, lines)
}

func TestEachBoardHasItsOwnNamesAndOneWithNoneYetIsLeftOut(t *testing.T) {
	boards := t.TempDir()
	write := func(path string) {
		t.Helper()
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(boards, path)), 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(boards, path), []byte("{}"), 0o600))
	}
	write("earth-1/landmassNames-0123abcd.json")
	write("earth-1/runs-0123abcd.bin")
	write("siege-1/landmassNames-4567cdef.json")
	write("earth-2/runs-89abcdef.bin")

	found, err := landmass_source.Boards(boards)

	require.NoError(t, err)
	assert.Equal(t, []landmass_source.Board{
		{Name: "earth-1", Path: filepath.Join(boards, "earth-1", "landmassNames-0123abcd.json")},
		{Name: "siege-1", Path: filepath.Join(boards, "siege-1", "landmassNames-4567cdef.json")},
	}, found)
}

func TestABoardWithTwoFilesOfNamesIsRefused(t *testing.T) {
	boards := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(boards, "earth-1"), 0o750))
	for _, name := range []string{"landmassNames-0123abcd.json", "landmassNames-4567cdef.json"} {
		require.NoError(t, os.WriteFile(filepath.Join(boards, "earth-1", name), []byte("{}"), 0o600))
	}

	_, err := landmass_source.Boards(boards)

	require.ErrorIs(t, err, landmass_source.ErrNamesTwice)
}
