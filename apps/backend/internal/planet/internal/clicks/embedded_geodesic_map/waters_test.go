package embedded_geodesic_map_test

import (
	"log/slog"
	"math"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mapdata "github.com/raphoester/clickplanet.lol-backend/generated/map"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/embedded_geodesic_map"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
)

func loader() *embedded_geodesic_map.Loader {
	return embedded_geodesic_map.New(tiles, slog.New(slog.DiscardHandler))
}

func TestTheShippedWatersAreTheBodiesTheLatticeDraws(t *testing.T) {
	waters, err := loader().LoadWaters()
	require.NoError(t, err)

	_, asset, err := mapdata.Waters()
	require.NoError(t, err)
	assert.Equal(t, "waters-eb8a3c8f.json", asset)

	require.Equal(t, 342, waters.Bodies())
	coasts := make([]int, 0, 4)
	for body := range 4 {
		coasts = append(coasts, len(waters.CoastOf(clicks.BodyID(body)))) //nolint:gosec // four bodies.
	}
	assert.Equal(t, []int{15294, 205, 161, 136}, coasts,
		"the ocean, the Black Sea cut off at the Bosporus, the Caspian, Michigan and Huron")

	coast, onTwo := 0, 0
	for tile := uint32(1); tile <= tiles; tile++ {
		bodies := waters.BodiesOf(tile)
		if len(bodies) == 0 {
			continue
		}
		coast++
		if len(bodies) > 1 {
			onTwo++
		}
		assert.Less(t, len(geography.Neighbours(tile)), clicks.MaxDegree,
			"tile %d is on a coast, so one of its lattice neighbours is water", tile)
	}
	assert.Equal(t, 18839, coast)
	assert.Equal(t, 385, onTwo)
}

func TestTheShippedCapitalsAreEachCountrysTileNearestItsCapitalCity(t *testing.T) {
	borders, err := loader().LoadBorders()
	require.NoError(t, err)
	capitals, err := loader().LoadCapitals(borders)
	require.NoError(t, err)

	_, asset, err := mapdata.Capitals()
	require.NoError(t, err)
	assert.Equal(t, "capitals-3938708b.json", asset)

	assert.Len(t, capitals.All(), 239, "every country on the map")
	flags := capitals.Flags(cpcountries.New().CheckCountry).All()
	assert.Len(t, flags, 235, "hm, nc, pm and sh have ground and no flag")

	paris := slices.IndexFunc(flags, func(capital clicks.Capital) bool { return capital.Country == "fr" })
	require.GreaterOrEqual(t, paris, 0)
	at, ok := geography.Position(flags[paris].Tile)
	require.True(t, ok)
	assert.Less(t, kilometres(at, 48.857, 2.352), 25.0, "a tile is 25 km across")
}

func TestCapitalsOffTheirGroundRefuseTheBoot(t *testing.T) {
	_, err := loader().LoadCapitals(clicks.BordersOf(tiles))
	require.ErrorContains(t, err, "ground")
}

// kilometres from a tile to a point, on the lattice's axes: x = -cos(lat)cos(lon), y = sin(lat), z = cos(lat)sin(lon).
func kilometres(at clicks.Vec3, lat, lon float64) float64 {
	phi, lambda := lat*math.Pi/180, lon*math.Pi/180
	point := clicks.Vec3{X: -math.Cos(phi) * math.Cos(lambda), Y: math.Sin(phi), Z: math.Cos(phi) * math.Sin(lambda)}
	unit := at.Unit()
	return math.Acos(min(1, unit.X*point.X+unit.Y*point.Y+unit.Z*point.Z)) * 6371
}
