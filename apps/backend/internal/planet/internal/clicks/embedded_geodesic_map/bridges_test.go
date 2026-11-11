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

func TestTheShippedBridgesLandOnCoastsAndJoinEveryPieceOfLand(t *testing.T) {
	bridges, err := loader().LoadBridges()
	require.NoError(t, err)

	_, asset, err := mapdata.Bridges()
	require.NoError(t, err)
	assert.Equal(t, "bridges-f06b9129.json", asset)
	assert.Equal(t, 582, bridges.Count(), "565 stepping stones and 17 sea lanes")

	for tile := uint32(1); tile <= tiles; tile++ {
		if len(bridges.Across(tile)) > 0 {
			assert.Less(t, len(geography.Neighbours(tile)), clicks.MaxDegree,
				"tile %d is a bridge's end, so it is on a coast and one of its lattice neighbours is water", tile)
		}
	}

	reached := make([]bool, tiles+1)
	reached[1] = true
	queue := []uint32{1}
	for len(queue) > 0 {
		tile := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		for _, next := range append(slices.Clone(geography.Neighbours(tile)), bridges.Across(tile)...) {
			if !reached[next] {
				reached[next] = true
				queue = append(queue, next)
			}
		}
	}
	assert.NotContains(t, reached[1:], false, "every tile is reached over land and bridges")
}

func TestTheShippedBridgesCrossTheNorthAtlanticAndReachHawaii(t *testing.T) {
	bridges, err := loader().LoadBridges()
	require.NoError(t, err)

	crosses := func(fromLat, fromLon, toLat, toLon float64) bool {
		for tile := uint32(1); tile <= tiles; tile++ {
			from, _ := geography.Position(tile)
			if kilometres(from, fromLat, fromLon) > 150 {
				continue
			}
			for _, end := range bridges.Across(tile) {
				to, _ := geography.Position(end)
				if kilometres(to, toLat, toLon) < 150 {
					return true
				}
			}
		}
		return false
	}

	assert.True(t, crosses(62.0, -6.8, 64.5, -14.0), "the Faroes to Iceland")
	assert.True(t, crosses(66.3, -22.2, 68.8, -26.0), "Iceland to Greenland")
	assert.True(t, crosses(21.4, -158.0, 37.7, -122.5), "Hawaii to California")
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
