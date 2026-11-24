package regions_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/cmd/flagpins/internal/regions"
)

// Two squares of one country: ENG from 0 to 10, SCT from 10 to 20 in latitude, both 0 to 10 in longitude.
func subunits(t *testing.T) regions.Layer {
	t.Helper()
	square := func(part string, from float64) regions.Feature {
		coordinates, err := json.Marshal([][][]float64{{{0, from}, {10, from}, {10, from + 10}, {0, from + 10}, {0, from}}})
		require.NoError(t, err)
		return regions.Feature{
			Properties: map[string]any{"ISO_A2_EH": "GB", "SU_A3": part},
			Geometry:   regions.Geometry{Type: "Polygon", Coordinates: coordinates},
		}
	}
	return regions.Layer{Features: []regions.Feature{square("ENG", 0), square("SCT", 10)}}
}

func place(name, class string, capital float64, pop float64, latitude, longitude float64) regions.Feature {
	return regions.Feature{Properties: map[string]any{
		"ISO_A2": "GB", "NAME": name, "FEATURECLA": class, "ADM0CAP": capital, "CAPALT": 0.0, "POP_MAX": pop,
		"LATITUDE": latitude, "LONGITUDE": longitude,
	}}
}

func TestEachRegionTakesTheCapitalInsideItsSubunit(t *testing.T) {
	places := regions.Layer{Features: []regions.Feature{
		place("Manchester", "Populated place", 0, 2_000_000, 5, 6),
		place("London", "Admin-0 capital", 1, 8_000_000, 2, 3),
		place("Glasgow", "Populated place", 0, 1_100_000, 14, 2),
		place("Edinburgh", "Admin-0 region capital", 0, 500_000, 15, 4),
	}}

	pins, err := regions.Pins([]string{"fr", "gb-eng", "gb-sct"}, subunits(t), places)

	require.NoError(t, err)
	assert.Equal(t, map[string]regions.Pin{
		"gb-eng": {Place: "London", Latitude: 2, Longitude: 3},
		"gb-sct": {Place: "Edinburgh", Latitude: 15, Longitude: 4},
	}, pins, "a country's code is no region's")
}

func TestARegionWithNoCapitalTakesItsBiggestPlace(t *testing.T) {
	places := regions.Layer{Features: []regions.Feature{
		place("Glasgow", "Populated place", 0, 1_100_000, 14, 2),
		place("Aberdeen", "Populated place", 0, 200_000, 17, 8),
	}}

	pins, err := regions.Pins([]string{"gb-sct"}, subunits(t), places)

	require.NoError(t, err)
	assert.Equal(t, "Glasgow", pins["gb-sct"].Place)
}

func TestARegionWithNoPlaceIsRefused(t *testing.T) {
	_, err := regions.Pins([]string{"gb-eng"}, subunits(t), regions.Layer{})

	require.ErrorIs(t, err, regions.ErrNoCapital)
}

func TestARegionWithNoSubunitIsRefused(t *testing.T) {
	_, err := regions.Pins([]string{"gb-wls"}, subunits(t), regions.Layer{})

	assert.ErrorContains(t, err, "no map subunit WLS of GB")
}
