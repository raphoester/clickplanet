package cpflags

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEachRegionTheGameShipsWithHasItsCapital(t *testing.T) {
	assert.Equal(t, []BuiltInPin{
		{Code: "gb-eng", Latitude: 51.501941, Longitude: -0.118668},
		{Code: "gb-sct", Latitude: 55.948328, Longitude: -3.219091},
		{Code: "gb-wls", Latitude: 51.499995, Longitude: -3.225008},
	}, Pins())
}

func TestAPinOfNoFlagTheGameShipsWithIsRefused(t *testing.T) {
	_, err := pinsOf([]byte(`{"xb": {"latitude": 48.1, "longitude": -1.7}}`))

	require.ErrorContains(t, err, `"xb" is no flag the game ships with`)
}

func TestAPinOffTheGlobeIsRefused(t *testing.T) {
	_, err := pinsOf([]byte(`{"gb-eng": {"latitude": 91, "longitude": 0}}`))

	require.ErrorContains(t, err, "off the globe")
}
