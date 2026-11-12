package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/cmd/translate/internal/flags"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpanthropic"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation/natural_earth"
)

func page(t *testing.T, french string) string {
	t.Helper()

	dir := t.TempDir()
	write := func(path, content string) {
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(dir, path)), 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(dir, path), []byte(content), 0o600))
	}
	write("src/i18n/locales/en.json", `{"chat.send": "Send", "chat.title": "Chat"}`)
	write("src/i18n/locales/fr.json", french)
	write("src/i18n/landmasses/fr.json", `{"264": "Corse"}`)
	write("src/app/Chat.tsx", `t("chat.send")`)
	write("static/landmassNames.json", `{"borders": "b.bin", "names": {"264": "Corsica"}}`)
	write("static/countries/countries.json", `{"ar": "Argentina"}`)
	write("src/i18n/countries/en.json", `{"in.ar": "in {flag} Argentina"}`)
	write("src/i18n/countries/fr.json", `{"in.ar": "en {flag} Argentine"}`)
	for _, layer := range []string{
		"ne_10m_geography_regions_polys", "ne_10m_geography_regions_points", "ne_10m_admin_0_map_subunits",
		"ne_50m_admin_0_map_subunits", "ne_10m_admin_1_states_provinces",
	} {
		write(filepath.Join("node_modules/.cache/natural-earth", natural_earth.Tag, layer+".geojson"), `{"features": []}`)
	}
	return dir
}

func TestThePlanSaysWhatWouldBeSentAndAsksNothing(t *testing.T) {
	t.Setenv(flags.KeyVariable, "")
	var out bytes.Buffer

	require.NoError(t, run(t.Context(), []string{"-page", page(t, `{"chat.send": "Envoyer"}`), "-plan"}, &out))

	assert.Equal(t, "lines fr: 1 to send, 0 kept, 1 written by hand, 0 no longer in English\n"+
		"  chat.title: \"Chat\"\n"+
		"landmasses fr: 0 to send, 0 kept, 1 written by hand, 0 no longer in English\n"+
		"countries en: 0 to send, 0 kept, 1 written by hand, 0 no longer in English\n"+
		"countries fr: 0 to send, 0 kept, 1 written by hand, 0 no longer in English\n", out.String())
}

func TestLinesToSendNeedAKey(t *testing.T) {
	t.Setenv(flags.KeyVariable, "")

	err := run(t.Context(), []string{"-page", page(t, `{"chat.send": "Envoyer"}`)}, &bytes.Buffer{})

	require.ErrorIs(t, err, cpanthropic.ErrNoKey)
}

func TestLinesWrittenByHandAreFingerprintedWithNoKey(t *testing.T) {
	t.Setenv(flags.KeyVariable, "")
	dir := page(t, `{"chat.send": "Envoyer", "chat.title": "Discussion"}`)
	var out bytes.Buffer

	require.NoError(t, run(t.Context(), []string{"-page", dir}, &out))

	fingerprints, err := os.ReadFile(filepath.Join(dir, "src/i18n/fingerprints/fr.json")) //nolint:gosec // the test's own folder.
	require.NoError(t, err)
	assert.JSONEq(t, fmt.Sprintf(`{"chat.send": %q, "chat.title": %q}`, translation.FingerprintOf("Send"), translation.FingerprintOf("Chat")), string(fingerprints))
	landmasses, err := os.ReadFile(filepath.Join(dir, "src/i18n/fingerprints/landmasses/fr.json")) //nolint:gosec // the test's own folder.
	require.NoError(t, err)
	assert.Contains(t, string(landmasses), `"264"`)
	countries, err := os.ReadFile(filepath.Join(dir, "src/i18n/fingerprints/countries/en.json")) //nolint:gosec // the test's own folder.
	require.NoError(t, err)
	assert.JSONEq(t, fmt.Sprintf(`{"in.ar": %q}`, translation.FingerprintOf("in {flag} Argentina")), string(countries))
	assert.Contains(t, out.String(), "lines fr: 0 translated, 0 kept, 2 written by hand")
}
