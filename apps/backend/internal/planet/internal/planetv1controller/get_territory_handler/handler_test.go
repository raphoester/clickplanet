package get_territory_handler_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_territory_handler"
)

type stubQuery struct {
	answer *planetv1.GetTerritoryResponse
}

func (s stubQuery) Territory() *planetv1.GetTerritoryResponse {
	return s.answer
}

func TestTheTerritoryIsTheQuerys(t *testing.T) {
	query := stubQuery{answer: &planetv1.GetTerritoryResponse{
		Tiles: 100, Territories: []*planetv1.Territory{{CountryId: "fr", Tiles: 3}},
	}}

	res, err := get_territory_handler.New(query).GetTerritory(t.Context(),
		connect.NewRequest(&planetv1.GetTerritoryRequest{}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(query.answer, res.Msg))
}
