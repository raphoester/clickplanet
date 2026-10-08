package territories_query_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_territories_handler/territories_query"
)

type stubTerritories map[string]uint32

func (s stubTerritories) Territories() map[string]uint32 {
	return s
}

func TestTheTerritoryIsTheMapsSizeAndWhatEachCountryHoldsInCountryOrder(t *testing.T) {
	query := territories_query.NewMemoryQuery(stubTerritories{"fr": 3, "bg": 7, "de": 1}, 100)

	assert.True(t, proto.Equal(&planetv1.GetTerritoriesResponse{
		Tiles: 100,
		Territories: []*planetv1.Territory{
			{CountryId: "bg", Tiles: 7},
			{CountryId: "de", Tiles: 1},
			{CountryId: "fr", Tiles: 3},
		},
	}, query.Territories()))
}

func TestAnEmptyMapIsItsSizeAlone(t *testing.T) {
	query := territories_query.NewMemoryQuery(stubTerritories{}, 100)

	assert.True(t, proto.Equal(&planetv1.GetTerritoriesResponse{Tiles: 100}, query.Territories()))
}
