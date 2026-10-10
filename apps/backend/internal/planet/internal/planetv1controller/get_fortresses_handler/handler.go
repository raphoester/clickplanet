package get_fortresses_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

type Query interface {
	Fortresses() *planetv1.GetFortressesResponse
}

func New(query Query) GetFortressesHandler {
	return GetFortressesHandler{query: query}
}

type GetFortressesHandler struct {
	query Query
}

func (h GetFortressesHandler) GetFortresses(
	_ context.Context,
	_ *connect.Request[planetv1.GetFortressesRequest],
) (*connect.Response[planetv1.GetFortressesResponse], error) {
	return connect.NewResponse(h.query.Fortresses()), nil
}
