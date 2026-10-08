package get_territories_handler_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_territories_handler"
)

type stubQuery struct {
	answer *planetv1.GetTerritoriesResponse
}

func (s stubQuery) Territories() *planetv1.GetTerritoriesResponse {
	return s.answer
}

func TestTheTerritoryIsTheQuerys(t *testing.T) {
	query := stubQuery{answer: &planetv1.GetTerritoriesResponse{
		Tiles: 100, Territories: []*planetv1.Territory{{CountryId: "fr", Tiles: 3}},
	}}

	res, err := get_territories_handler.New(query).GetTerritories(t.Context(),
		connect.NewRequest(&planetv1.GetTerritoriesRequest{}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(query.answer, res.Msg))
}
