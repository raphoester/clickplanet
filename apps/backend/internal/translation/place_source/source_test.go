package place_source_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/translation"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation/place_source"
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

	lines, order, err := place_source.Read(path, known)

	require.NoError(t, err)
	assert.Equal(t, []translation.Key{"20", "13", "9"}, order)
	assert.Equal(t, []translation.Line{
		{Key: "20", English: "Ionian Islands 2", Context: `"2" tells it from another place named "Ionian Islands": keep it. Natural Earth names it: de "Ionische Inseln", fr "Îles Ioniennes".`},
		{Key: "13", English: "Split-Dalmatia", Context: `Natural Earth names it: fr "Split-Dalmatie".`},
		{Key: "9", English: "Atlantis"},
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

	found, err := place_source.Boards(boards, place_source.Landmasses)

	require.NoError(t, err)
	assert.Equal(t, []place_source.Board{
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

	_, err := place_source.Boards(boards, place_source.Landmasses)

	require.ErrorIs(t, err, place_source.ErrNamesTwice)
}

func TestABoardTakesTheNameAnotherGivesTheSameEnglishWithItsNumber(t *testing.T) {
	names := place_source.Names{}
	names.Learn([]translation.Line{{Key: "1", English: "Crete"}, {Key: "2", English: "Ionian Islands"}, {Key: "3", English: "Zealand"}}, translation.Catalogue{
		Lines: map[translation.Key]string{"1": "Crète", "2": "Îles Ioniennes", "3": "Seeland"},
		Fingerprints: map[translation.Key]translation.Fingerprint{
			"1": translation.FingerprintOf("Crete"), "3": translation.FingerprintOf("Sealand"),
		},
	})
	names.Learn([]translation.Line{{Key: "8", English: "Crete"}}, translation.Catalogue{
		Lines: map[translation.Key]string{"8": "Candie"}, Fingerprints: map[translation.Key]translation.Fingerprint{},
	})

	for english, want := range map[string]string{"Crete": "Crète", "Crete 3": "Crète 3", "Ionian Islands 2": "Îles Ioniennes 2"} {
		got, ok := names.Recall(english)
		assert.True(t, ok, english)
		assert.Equal(t, want, got)
	}
	for _, english := range []string{"Zealand", "Atlantis", "Atlantis 2", "Crete Island"} {
		_, ok := names.Recall(english)
		assert.False(t, ok, english)
	}
}

func TestAProvinceTakesNaturalEarthsNameInItsLanguagesWithItsNumber(t *testing.T) {
	known := gazetteer{"Bavaria": {"fr": "Bavière", "de": "Bayern"}, "Isfahan": {"tr": "İsfahan Eyaleti", "fr": "Ispahan"}}

	for english, want := range map[string]string{"Bavaria": "Bavière", "Bavaria 2": "Bavière 2"} {
		got, ok := place_source.NaturalEarth{Gazetteer: known, Language: "fr"}.Recall(english)
		assert.True(t, ok, english)
		assert.Equal(t, want, got)
	}
	_, ok := place_source.NaturalEarth{Gazetteer: known, Language: "cs"}.Recall("Bavaria")
	assert.False(t, ok, "Natural Earth has no Czech")
	_, ok = place_source.NaturalEarth{Gazetteer: known, Language: "fr"}.Recall("Central Germany")
	assert.False(t, ok)
	_, ok = place_source.NaturalEarth{Gazetteer: known, Language: "tr"}.Recall("Isfahan")
	assert.False(t, ok, "a word for the kind of division the English does not have")
}

func TestEachBoardsProvincesAreNamedInAFileOfTheirOwn(t *testing.T) {
	boards := t.TempDir()
	for _, path := range []string{"earth-1/landmassNames-0123abcd.json", "earth-2/landmassNames-4567cdef.json", "earth-2/provinceNames-89abcdef.json"} {
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(boards, path)), 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(boards, path), []byte("{}"), 0o600))
	}

	found, err := place_source.Boards(boards, place_source.Provinces)

	require.NoError(t, err)
	assert.Equal(t, []place_source.Board{{Name: "earth-2", Path: filepath.Join(boards, "earth-2", "provinceNames-89abcdef.json")}}, found)
}
