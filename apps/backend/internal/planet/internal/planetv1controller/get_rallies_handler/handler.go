package get_rallies_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

// Query is every rally still on, of every flag.
type Query interface {
	Rallies(ctx context.Context) (*planetv1.GetRalliesResponse, error)
}

func New(query Query) GetRalliesHandler {
	return GetRalliesHandler{query: query}
}

type GetRalliesHandler struct {
	query Query
}

func (h GetRalliesHandler) GetRallies(
	ctx context.Context,
	_ *connect.Request[planetv1.GetRalliesRequest],
) (*connect.Response[planetv1.GetRalliesResponse], error) {
	rallies, err := h.query.Rallies(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the query names it, and the error net answers it.
	}

	return connect.NewResponse(rallies), nil
}
