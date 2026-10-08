package get_territories_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

type Query interface {
	Territories() *planetv1.GetTerritoriesResponse
}

func New(query Query) GetTerritoriesHandler {
	return GetTerritoriesHandler{query: query}
}

type GetTerritoriesHandler struct {
	query Query
}

func (h GetTerritoriesHandler) GetTerritories(
	_ context.Context,
	_ *connect.Request[planetv1.GetTerritoriesRequest],
) (*connect.Response[planetv1.GetTerritoriesResponse], error) {
	return connect.NewResponse(h.query.Territories()), nil
}
