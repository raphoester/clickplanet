package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/translation"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation/page_catalogues"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path) //nolint:gosec // the test's own folder.
	require.NoError(t, err)
	return string(data)
}

func page(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write(t, filepath.Join(dir, "src/i18n/locales/en.json"), `{"chat.send": "Send", "chat.title": "Chat", "chat.took": "{player} took it"}`)
	write(t, filepath.Join(dir, "src/i18n/locales/fr.json"), `{"chat.send": "Envoyer"}`)
	write(t, filepath.Join(dir, "src/app/Chat.tsx"), `t("chat.send")`)
	write(t, filepath.Join(dir, "static/boards/earth-1/landmassNames-0123abcd.json"), `{"borders": "b.bin", "names": {"264": "Corsica"}}`)
	write(t, filepath.Join(dir, "src/i18n/landmasses/earth-1/fr.json"), `{"264": "Corse"}`)
	write(t, filepath.Join(dir, "static/boards/earth-2/landmassNames-4567cdef.json"), `{"borders": "b.bin", "names": {"901": "Corsica 2", "902": "Atlantis"}}`)
	write(t, filepath.Join(dir, "static/countries/countries.json"), `{"ar": "Argentina"}`)
	write(t, filepath.Join(dir, "src/i18n/countries/en.json"), `{"in.ar": "in {flag} Argentina"}`)
	write(t, filepath.Join(dir, "src/i18n/countries/fr.json"), `{"in.ar": "en {flag} Argentine"}`)
	return dir
}

func TestThePlanWritesTheLinesEachLanguageLacksInBatches(t *testing.T) {
	dir := page(t)
	work := filepath.Join(t.TempDir(), "work")
	var out bytes.Buffer

	require.NoError(t, run([]string{"plan", "-page", dir, "-work", work}, &out))

	assert.Contains(t, out.String(), "lines fr: 2 to translate, 0 kept, 1 written by hand, 0 named on another board, 0 no longer in English\n")
	assert.Contains(t, out.String(), "landmasses/earth-2 fr: 1 to translate, 0 kept, 0 written by hand, 1 named on another board")
	assert.Contains(t, out.String(), "Natural Earth: the page's cache lacks ne_10m_geography_regions_polys")
	assert.JSONEq(t, `{"catalogue": "lines", "language": "fr", "lines": [
		{"key": "chat.title", "english": "Chat"},
		{"key": "chat.took", "english": "{player} took it"}
	]}`, read(t, filepath.Join(work, "lines", "fr.1.json")))
	assert.JSONEq(t, `{"catalogue": "landmasses/earth-2", "language": "fr", "lines": [{"key": "902", "english": "Atlantis"}]}`,
		read(t, filepath.Join(work, "landmasses", "earth-2", "fr.1.json")))
	assert.NoFileExists(t, filepath.Join(work, "countries", "fr.1.json"))
}

func TestABatchHoldsNoMoreLinesThanAsked(t *testing.T) {
	work := filepath.Join(t.TempDir(), "work")

	require.NoError(t, run([]string{"plan", "-page", page(t), "-work", work, "-batch", "1"}, &bytes.Buffer{}))

	assert.FileExists(t, filepath.Join(work, "lines", "fr.1.json"))
	assert.FileExists(t, filepath.Join(work, "lines", "fr.2.json"))
}

func TestAPlanRefusesAFolderInUse(t *testing.T) {
	work := t.TempDir()
	write(t, filepath.Join(work, "lines", "fr.1.answers.json"), `{}`)

	err := run([]string{"plan", "-page", page(t), "-work", work}, &bytes.Buffer{})

	require.ErrorIs(t, err, page_catalogues.ErrFolderInUse)
}

func TestApplyWritesTheAnswersThatKeepTheirSlotsWithTheirFingerprints(t *testing.T) {
	dir := page(t)
	work := t.TempDir()
	write(t, filepath.Join(work, "lines", "fr.1.answers.json"), `{"chat.title": "Discussion", "chat.took": "{joueur} l'a pris"}`)
	write(t, filepath.Join(work, "landmasses", "earth-2", "fr.1.answers.json"), `{"902": "Atlantide"}`)
	var out bytes.Buffer

	err := run([]string{"apply", "-page", dir, "-work", work}, &out)

	require.ErrorIs(t, err, errUntranslated)
	assert.Contains(t, out.String(), "  refused chat.took: slots joueur:value where the English has player:value\n")
	assert.JSONEq(t, `{"chat.send": "Envoyer", "chat.title": "Discussion"}`, read(t, filepath.Join(dir, "src/i18n/locales/fr.json")))
	assert.JSONEq(t, fmt.Sprintf(`{"chat.send": %q, "chat.title": %q}`, translation.FingerprintOf("Send"), translation.FingerprintOf("Chat")),
		read(t, filepath.Join(dir, "src/i18n/fingerprints/fr.json")))
	assert.JSONEq(t, `{"901": "Corse 2", "902": "Atlantide"}`, read(t, filepath.Join(dir, "src/i18n/landmasses/earth-2/fr.json")))
	assert.JSONEq(t, fmt.Sprintf(`{"in.ar": %q}`, translation.FingerprintOf("in {flag} Argentina")),
		read(t, filepath.Join(dir, "src/i18n/fingerprints/countries/en.json")))
}

func TestApplyWithEveryLineAnsweredSucceeds(t *testing.T) {
	dir := page(t)
	work := t.TempDir()
	write(t, filepath.Join(work, "lines", "fr.1.answers.json"), `{"chat.title": "Discussion"}`)
	write(t, filepath.Join(work, "lines", "fr.2.answers.json"), `{"chat.took": "{player} l'a pris"}`)
	write(t, filepath.Join(work, "landmasses", "earth-2", "fr.1.answers.json"), `{"902": "Atlantide"}`)

	require.NoError(t, run([]string{"apply", "-page", dir, "-work", work}, &bytes.Buffer{}))
	require.NoError(t, run([]string{"plan", "-page", dir, "-work", filepath.Join(work, "again")}, &bytes.Buffer{}))

	assert.NoDirExists(t, filepath.Join(work, "again"))
}
