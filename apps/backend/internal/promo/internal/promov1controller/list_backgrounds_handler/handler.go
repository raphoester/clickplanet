package list_backgrounds_handler

import (
	"context"

	"connectrpc.com/connect"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
)

type Query interface {
	Backgrounds(ctx context.Context) (*promov1.ListBackgroundsResponse, error)
}

func New(query Query) ListBackgroundsHandler {
	return ListBackgroundsHandler{query: query}
}

type ListBackgroundsHandler struct {
	query Query
}

func (h ListBackgroundsHandler) ListBackgrounds(
	ctx context.Context,
	_ *connect.Request[promov1.ListBackgroundsRequest],
) (*connect.Response[promov1.ListBackgroundsResponse], error) {
	listed, err := h.query.Backgrounds(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(listed), nil
}
