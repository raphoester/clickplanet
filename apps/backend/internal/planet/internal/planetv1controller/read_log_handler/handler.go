package read_log_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

type Query interface {
	Entries(ctx context.Context, from uint64, limit uint32) (*planetv1.ReadLogResponse, error)
}

func New(query Query) ReadLogHandler {
	return ReadLogHandler{query: query}
}

type ReadLogHandler struct {
	query Query
}

func (h ReadLogHandler) ReadLog(
	ctx context.Context,
	req *connect.Request[planetv1.ReadLogRequest],
) (*connect.Response[planetv1.ReadLogResponse], error) {
	entries, err := h.query.Entries(ctx, req.Msg.GetFromPosition(), req.Msg.GetLimit())
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}

	res := connect.NewResponse(entries)
	res.Header().Set("Cache-Control", "no-store")

	return res, nil
}
