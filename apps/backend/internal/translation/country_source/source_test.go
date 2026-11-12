package country_source_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/translation"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation/country_source"
)

func TestEachFlagIsALineOfItsCountryInASentence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "countries.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"ar": "Argentina", "us": "United States"}`), 0o600))

	lines, order, err := country_source.Read(path)

	require.NoError(t, err)
	assert.Equal(t, []translation.Key{"in.ar", "in.us"}, order)
	require.Len(t, lines, 2)
	assert.Equal(t, translation.Key("in.us"), lines[1].Key)
	assert.Equal(t, "in {flag} United States", lines[1].English)
	assert.Contains(t, lines[1].Context, `The country's ISO code is "us".`)
}
