package geodesic_map_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/secondary/geodesic_map"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func TestTheShippedBordersPutEveryTileInItsCountry(t *testing.T) {
	borders, asset, err := geodesic_map.LoadBorders(tiles)
	require.NoError(t, err)

	assert.Equal(t, "borders-e9353d0c.bin", asset)

	counts := map[string]int{}
	for tile := uint32(1); tile <= tiles; tile++ {
		counts[borders.CountryOf(tile)]++
	}

	assert.Len(t, counts, 190, "189 countries and no country")
	assert.Equal(t, 2191, counts[""])
	assert.Equal(t, 1239, counts["fr"])
	assert.Equal(t, "fr", borders.CountryOf(99290))
	assert.Empty(t, borders.CountryOf(0), "tile ids start at 1")
	assert.Empty(t, borders.CountryOf(tiles+1))
}

func TestBordersForAnotherMapRefuseTheBoot(t *testing.T) {
	_, _, err := geodesic_map.LoadBorders(tiles - 1)
	require.Error(t, err)
}

func TestARegionPastTheCodeTableIsRefused(t *testing.T) {
	_, err := clicks.NewBorders([]uint16{0, 2}, []string{"", "fr"})
	require.Error(t, err)
}
