package get_feed_start_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

type Query interface {
	Start(ctx context.Context) (*planetv1.GetFeedStartResponse, error)
}

func New(query Query) GetFeedStartHandler {
	return GetFeedStartHandler{query: query}
}

type GetFeedStartHandler struct {
	query Query
}

func (h GetFeedStartHandler) GetFeedStart(
	ctx context.Context,
	_ *connect.Request[planetv1.GetFeedStartRequest],
) (*connect.Response[planetv1.GetFeedStartResponse], error) {
	start, err := h.query.Start(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(start), nil
}
