package territories_query_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_territories_handler/territories_query"
)

type stubTally clicks.Tally

func (s stubTally) Tally() (clicks.MapVersion, clicks.Tally) {
	return clicks.MapVersion{}, clicks.Tally(s)
}

type stubShields []clicks.FlagShields

func (s stubShields) ShieldTally() []clicks.FlagShields {
	return s
}

type stubBorders map[clicks.LandmassID]string

func (s stubBorders) CountryOfLandmass(landmass clicks.LandmassID) string {
	return s[landmass]
}

func (stubBorders) Name() clicks.BoardName { return "earth-1" }

func (stubBorders) Tiles() uint32 { return 100 }

// earth1 is earth-1's size.
type earth1 struct{ stubBorders }

func (earth1) Tiles() uint32 { return 260_504 }

func TestTheTerritoryIsTheMapsSizeAndWhatEachCountryHoldsInCountryOrder(t *testing.T) {
	query := territories_query.NewMemoryQuery(stubTally{
		Countries: []clicks.FlagTiles{{Flag: "fr", Tiles: 3}, {Flag: "bg", Tiles: 7}, {Flag: "de", Tiles: 1}},
	}, stubShields{}, stubBorders{})

	assert.True(t, proto.Equal(&planetv1.GetTerritoriesResponse{
		Tiles: 100, Board: "earth-1",
		Territories: []*planetv1.Territory{
			{CountryId: "bg", Tiles: 7},
			{CountryId: "de", Tiles: 1},
			{CountryId: "fr", Tiles: 3},
		},
	}, query.Territories()))
}

func TestATerritorySaysHowManyTilesOfEachCountrysLandTheFlagHoldsInCountryOrder(t *testing.T) {
	query := territories_query.NewMemoryQuery(stubTally{
		Countries: []clicks.FlagTiles{{Flag: "fr", Tiles: 5}, {Flag: "it", Tiles: 2}},
		Landmasses: []clicks.LandmassTiles{
			{Landmass: 1, Flags: []clicks.FlagTiles{{Flag: "fr", Tiles: 1}}},
			{Landmass: 2, Flags: []clicks.FlagTiles{{Flag: "fr", Tiles: 2}, {Flag: "it", Tiles: 2}}},
			{Landmass: 3, Flags: []clicks.FlagTiles{{Flag: "fr", Tiles: 2}}},
		},
	}, stubShields{}, stubBorders{1: "it", 2: "fr", 3: "fr"})

	assert.True(t, proto.Equal(&planetv1.GetTerritoriesResponse{
		Tiles: 100, Board: "earth-1",
		Territories: []*planetv1.Territory{
			{CountryId: "fr", Tiles: 5, Grounds: []*planetv1.Ground{{CountryId: "fr", Tiles: 4}, {CountryId: "it", Tiles: 1}}},
			{CountryId: "it", Tiles: 2, Grounds: []*planetv1.Ground{{CountryId: "fr", Tiles: 2}}},
		},
	}, query.Territories()), "two landmasses of one country add up")
}

func TestATerritorySaysTheShieldsOnItsTiles(t *testing.T) {
	query := territories_query.NewMemoryQuery(stubTally{
		Countries: []clicks.FlagTiles{{Flag: "pl", Tiles: 25_862}, {Flag: "sc", Tiles: 18_272}},
	}, stubShields{{Flag: "pl", Shields: 18_726}}, earth1{})

	assert.True(t, proto.Equal(&planetv1.GetTerritoriesResponse{
		Tiles: 260_504, Board: "earth-1",
		Territories: []*planetv1.Territory{
			{CountryId: "pl", Tiles: 25_862, Shields: 18_726},
			{CountryId: "sc", Tiles: 18_272},
		},
	}, query.Territories()), "a flag the tally names no shields of has none")
}

func TestATileOnNoCountrysLandIsOnNoGround(t *testing.T) {
	query := territories_query.NewMemoryQuery(stubTally{
		Countries:  []clicks.FlagTiles{{Flag: "fr", Tiles: 1}},
		Landmasses: []clicks.LandmassTiles{{Landmass: clicks.NoLandmass, Flags: []clicks.FlagTiles{{Flag: "fr", Tiles: 1}}}},
	}, stubShields{}, stubBorders{})

	assert.True(t, proto.Equal(&planetv1.GetTerritoriesResponse{
		Tiles: 100, Board: "earth-1",
		Territories: []*planetv1.Territory{{CountryId: "fr", Tiles: 1}},
	}, query.Territories()))
}

func TestAnEmptyMapIsItsSizeAlone(t *testing.T) {
	query := territories_query.NewMemoryQuery(stubTally{}, stubShields{}, stubBorders{})

	assert.True(t, proto.Equal(&planetv1.GetTerritoriesResponse{Tiles: 100, Board: "earth-1"}, query.Territories()))
}
