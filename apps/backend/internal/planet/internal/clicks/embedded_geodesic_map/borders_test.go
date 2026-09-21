package embedded_geodesic_map_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mapdata "github.com/raphoester/clickplanet.lol-backend/generated/map"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/embedded_geodesic_map"
)

func TestTheShippedBordersPutEveryTileInItsCountry(t *testing.T) {
	borders, err := embedded_geodesic_map.New(tiles, nil).LoadBorders()
	require.NoError(t, err)

	_, asset, err := mapdata.Borders()
	require.NoError(t, err)

	assert.Equal(t, "borders-67e352b1.bin", asset)

	counts := map[string]int{}
	for tile := uint32(1); tile <= tiles; tile++ {
		counts[borders.CountryOf(tile)]++
	}

	// Both blobs come from one query now, so a tile exists exactly where a country does. Was 2,191.
	assert.NotContains(t, counts, "", "every tile is in a country")
	assert.Len(t, counts, 205, "countries with at least one tile")
	assert.Equal(t, 1248, counts["fr"])
	assert.Equal(t, "fr", borders.CountryOf(100336))
	assert.Equal(t, 22885, counts["aq"], "includes the ice shelves, which are in no country polygon")
	assert.Empty(t, borders.CountryOf(0), "tile ids start at 1")
	assert.Empty(t, borders.CountryOf(tiles+1))
}

func TestBordersForAnotherMapRefuseTheBoot(t *testing.T) {
	_, err := embedded_geodesic_map.New(tiles-1, nil).LoadBorders()
	require.Error(t, err)
}

func TestARegionPastTheCodeTableIsRefused(t *testing.T) {
	_, err := clicks.NewBorders([]uint16{0, 2}, []string{"", "fr"})
	require.Error(t, err)
}
