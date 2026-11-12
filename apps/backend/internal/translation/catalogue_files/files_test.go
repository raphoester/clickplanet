package catalogue_files_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/translation"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation/catalogue_files"
)

func TestAFileKeepsItsKeysInOrderAndItsHTMLAsItIs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fr.json")
	lines := map[translation.Key]string{"b": "Lis les <terms>conditions</terms> & \"règles\"", "a": "Envoyer"}

	require.NoError(t, catalogue_files.Write(path, []translation.Key{"b", "a", "absent"}, lines))
	written, err := os.ReadFile(path) //nolint:gosec // the test's own folder.
	require.NoError(t, err)
	read, err := catalogue_files.Read(path)
	require.NoError(t, err)

	assert.Equal(t, []string{"{", `  "b": "Lis les <terms>conditions</terms> & \"règles\"",`, `  "a": "Envoyer"`, "}", ""}, strings.Split(string(written), "\n"))
	assert.Equal(t, catalogue_files.Keyed{Order: []translation.Key{"b", "a"}, Lines: lines}, read)
}

func TestAFileThatIsNotThereIsEmpty(t *testing.T) {
	read, err := catalogue_files.Read(filepath.Join(t.TempDir(), "none.json"))

	require.NoError(t, err)
	assert.Empty(t, read.Lines)
}

func TestTheLanguagesAreTheFilesBesideTheEnglish(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"en.json", "fr.json", "pt-BR.json", "notes.txt"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0o600))
	}
	place := catalogue_files.Place{English: filepath.Join(dir, "en.json"), Lines: dir, Fingerprints: filepath.Join(dir, "fingerprints")}

	languages, err := place.Languages()

	require.NoError(t, err)
	assert.Equal(t, []translation.Language{"fr", "pt-BR"}, languages)
}

func TestACatalogueIsSavedWithItsFingerprints(t *testing.T) {
	dir := t.TempDir()
	place := catalogue_files.Place{English: filepath.Join(dir, "en.json"), Lines: dir, Fingerprints: filepath.Join(dir, "fingerprints")}
	catalogue := translation.Catalogue{
		Lines:        map[translation.Key]string{"a": "Envoyer"},
		Fingerprints: map[translation.Key]translation.Fingerprint{"a": translation.FingerprintOf("Send")},
	}

	require.NoError(t, place.Save("fr", []translation.Key{"a"}, catalogue))
	read, err := place.Catalogue("fr")

	require.NoError(t, err)
	assert.Equal(t, catalogue, read)
}
