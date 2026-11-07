package list_graffiti_handler

import (
	"context"

	"connectrpc.com/connect"

	vandalsquadv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/vandalsquad/v1"
)

type Query interface {
	Graffiti() *vandalsquadv1.ListGraffitiResponse
}

func New(query Query) ListGraffitiHandler {
	return ListGraffitiHandler{query: query}
}

type ListGraffitiHandler struct {
	query Query
}

func (h ListGraffitiHandler) ListGraffiti(
	_ context.Context,
	_ *connect.Request[vandalsquadv1.ListGraffitiRequest],
) (*connect.Response[vandalsquadv1.ListGraffitiResponse], error) {
	return connect.NewResponse(h.query.Graffiti()), nil
}
