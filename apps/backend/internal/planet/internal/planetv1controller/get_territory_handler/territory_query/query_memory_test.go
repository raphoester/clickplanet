package territory_query_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_territory_handler/territory_query"
)

type stubTerritories map[string]uint32

func (s stubTerritories) Territories() map[string]uint32 {
	return s
}

func TestTheTerritoryIsTheMapsSizeAndWhatEachCountryHoldsInCountryOrder(t *testing.T) {
	query := territory_query.NewMemoryQuery(stubTerritories{"fr": 3, "bg": 7, "de": 1}, 100)

	assert.True(t, proto.Equal(&planetv1.GetTerritoryResponse{
		Tiles: 100,
		Territories: []*planetv1.Territory{
			{CountryId: "bg", Tiles: 7},
			{CountryId: "de", Tiles: 1},
			{CountryId: "fr", Tiles: 3},
		},
	}, query.Territory()))
}

func TestAnEmptyMapIsItsSizeAlone(t *testing.T) {
	query := territory_query.NewMemoryQuery(stubTerritories{}, 100)

	assert.True(t, proto.Equal(&planetv1.GetTerritoryResponse{Tiles: 100}, query.Territory()))
}
