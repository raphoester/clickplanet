package get_map_history_handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

type Query interface {
	History(ctx context.Context, since, until time.Time, step time.Duration) (*planetv1.MapHistory, error)
}

const mostFrames = 5000

var ErrInvalidWindow = errors.New("the window must start before it ends, with a step that makes 5,000 frames at most")

func New(query Query) GetMapHistoryHandler {
	return GetMapHistoryHandler{query: query}
}

type GetMapHistoryHandler struct {
	query Query
}

func (h GetMapHistoryHandler) GetMapHistory(
	ctx context.Context,
	req *connect.Request[planetv1.GetMapHistoryRequest],
) (*connect.Response[planetv1.GetMapHistoryResponse], error) {
	since, until, step := req.Msg.GetSince().AsTime(), req.Msg.GetUntil().AsTime(), req.Msg.GetStep().AsDuration()
	if req.Msg.GetSince() == nil || req.Msg.GetUntil() == nil || !since.Before(until) || step <= 0 ||
		until.Sub(since)/step >= mostFrames {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrInvalidWindow)
	}
	history, err := h.query.History(ctx, since, until, step)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&planetv1.GetMapHistoryResponse{History: history}), nil
}
