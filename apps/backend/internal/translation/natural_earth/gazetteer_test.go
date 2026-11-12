package natural_earth_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/translation/natural_earth"
)

func TestAPlaceIsNamedInEachLanguageWhereItsNamesAgree(t *testing.T) {
	folder := t.TempDir()
	write := func(name, features string) {
		require.NoError(t, os.WriteFile(filepath.Join(folder, name+".geojson"), []byte(`{"features": [`+features+`]}`), 0o600))
	}
	write("ne_10m_geography_regions_polys", `{"properties": {"NAME": "Crete", "NAME_FR": "Crète", "NAME_DE": "Kreta"}}`)
	write("ne_10m_geography_regions_points", ``)
	write("ne_10m_admin_0_map_subunits", ``)
	write("ne_50m_admin_0_map_subunits", ``)
	write("ne_10m_admin_1_states_provinces", `{"properties": {"name": "Northern", "name_fr": "Nord"}}, {"properties": {"name": "Northern", "name_fr": "Septentrionale", "name_de": "Nord"}}, {"properties": {"name_en": "Crete (Greece)", "name_fr": "Crète"}}`)

	gazetteer, err := natural_earth.Load(t.Context(), folder)

	require.NoError(t, err)
	assert.Equal(t, map[string]string{"fr": "Crète", "de": "Kreta"}, gazetteer.NamesOf("Crete"))
	assert.Equal(t, map[string]string{"de": "Nord"}, gazetteer.NamesOf("Northern"))
	assert.Empty(t, gazetteer.NamesOf("Atlantis"))
}
