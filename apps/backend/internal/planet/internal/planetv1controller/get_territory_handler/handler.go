package get_territory_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

type Query interface {
	Territory() *planetv1.GetTerritoryResponse
}

func New(query Query) GetTerritoryHandler {
	return GetTerritoryHandler{query: query}
}

type GetTerritoryHandler struct {
	query Query
}

func (h GetTerritoryHandler) GetTerritory(
	_ context.Context,
	_ *connect.Request[planetv1.GetTerritoryRequest],
) (*connect.Response[planetv1.GetTerritoryResponse], error) {
	return connect.NewResponse(h.query.Territory()), nil
}
