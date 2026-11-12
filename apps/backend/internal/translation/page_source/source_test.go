package page_source_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/translation"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation/page_source"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

func TestALineCarriesItsNoteAndTheCodeThatShowsIt(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "i18n", "locales", "en.json"), `{"settings.big": "Big", "chat.send": "Send"}`)
	write(t, filepath.Join(dir, "i18n", "notes.json"), `{"settings.big": "How the flags are drawn when zoomed out"}`)
	write(t, filepath.Join(dir, "app", "Settings.tsx"), "const VIEWS = [\n    {value: \"big\", label: \"settings.big\"},\n]")
	write(t, filepath.Join(dir, "app", "Settings.test.tsx"), `expect(t("settings.big")).toBe("Big")`)
	write(t, filepath.Join(dir, "i18n", "page.ts"), `t("chat.send")`)

	lines, order, err := page_source.Read(page_source.Files{
		English: filepath.Join(dir, "i18n", "locales", "en.json"),
		Notes:   filepath.Join(dir, "i18n", "notes.json"),
		Code:    dir,
	})

	require.NoError(t, err)
	assert.Equal(t, []translation.Key{"settings.big", "chat.send"}, order)
	assert.Equal(t, []translation.Line{
		{
			Key: "settings.big", English: "Big",
			Context: "Note: How the flags are drawn when zoomed out\n" + filepath.Join("app", "Settings.tsx") + `: {value: "big", label: "settings.big"},`,
		},
		{Key: "chat.send", English: "Send"},
	}, lines)
}
