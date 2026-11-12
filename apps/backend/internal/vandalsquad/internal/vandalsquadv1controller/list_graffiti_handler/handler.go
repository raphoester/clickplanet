package list_graffiti_handler

import (
	"context"

	"connectrpc.com/connect"

	vandalsquadv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/vandalsquad/v1"
)

type Query interface {
	Graffiti(ctx context.Context) (*vandalsquadv1.ListGraffitiResponse, error)
}

func New(query Query) ListGraffitiHandler {
	return ListGraffitiHandler{query: query}
}

type ListGraffitiHandler struct {
	query Query
}

func (h ListGraffitiHandler) ListGraffiti(
	ctx context.Context,
	_ *connect.Request[vandalsquadv1.ListGraffitiRequest],
) (*connect.Response[vandalsquadv1.ListGraffitiResponse], error) {
	graffiti, err := h.query.Graffiti(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(graffiti), nil
}
