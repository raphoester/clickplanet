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

type stubBorders map[clicks.LandmassID]string

func (s stubBorders) CountryOfLandmass(landmass clicks.LandmassID) string {
	return s[landmass]
}

func TestTheTerritoryIsTheMapsSizeAndWhatEachCountryHoldsInCountryOrder(t *testing.T) {
	query := territories_query.NewMemoryQuery(stubTally{
		Countries: []clicks.FlagTiles{{Flag: "fr", Tiles: 3}, {Flag: "bg", Tiles: 7}, {Flag: "de", Tiles: 1}},
	}, stubBorders{}, 100)

	assert.True(t, proto.Equal(&planetv1.GetTerritoriesResponse{
		Tiles: 100,
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
	}, stubBorders{1: "it", 2: "fr", 3: "fr"}, 100)

	assert.True(t, proto.Equal(&planetv1.GetTerritoriesResponse{
		Tiles: 100,
		Territories: []*planetv1.Territory{
			{CountryId: "fr", Tiles: 5, Grounds: []*planetv1.Ground{{CountryId: "fr", Tiles: 4}, {CountryId: "it", Tiles: 1}}},
			{CountryId: "it", Tiles: 2, Grounds: []*planetv1.Ground{{CountryId: "fr", Tiles: 2}}},
		},
	}, query.Territories()), "two landmasses of one country add up")
}

func TestATileOnNoCountrysLandIsOnNoGround(t *testing.T) {
	query := territories_query.NewMemoryQuery(stubTally{
		Countries:  []clicks.FlagTiles{{Flag: "fr", Tiles: 1}},
		Landmasses: []clicks.LandmassTiles{{Landmass: clicks.NoLandmass, Flags: []clicks.FlagTiles{{Flag: "fr", Tiles: 1}}}},
	}, stubBorders{}, 100)

	assert.True(t, proto.Equal(&planetv1.GetTerritoriesResponse{
		Tiles:       100,
		Territories: []*planetv1.Territory{{CountryId: "fr", Tiles: 1}},
	}, query.Territories()))
}

func TestAnEmptyMapIsItsSizeAlone(t *testing.T) {
	query := territories_query.NewMemoryQuery(stubTally{}, stubBorders{}, 100)

	assert.True(t, proto.Equal(&planetv1.GetTerritoriesResponse{Tiles: 100}, query.Territories()))
}
