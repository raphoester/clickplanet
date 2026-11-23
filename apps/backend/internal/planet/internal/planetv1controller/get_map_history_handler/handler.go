package get_map_history_handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/forgetting"
)

type Query interface {
	History(ctx context.Context, since, until time.Time, step time.Duration) (*planetv1.MapHistory, error)
}

type Forgettings interface {
	Forgetting(ctx context.Context) (forgetting.Forgetting, error)
}

const mostFrames = 5000

var ErrInvalidWindow = errors.New("the window must start before it ends, with a step that makes 5,000 frames at most")

func New(query Query, forgettings Forgettings) GetMapHistoryHandler {
	return GetMapHistoryHandler{query: query, forgettings: forgettings}
}

type GetMapHistoryHandler struct {
	query       Query
	forgettings Forgettings
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
	// Read before the history, so a revert during the read changes what the next call compares.
	forgotten, err := h.forgettings.Forgetting(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	if kept := req.Msg.KeptForgetting; kept != nil && forgetting.Forgetting(*kept) == forgotten {
		return connect.NewResponse(&planetv1.GetMapHistoryResponse{Forgetting: string(forgotten), Kept: true}), nil
	}
	history, err := h.query.History(ctx, since, until, step)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&planetv1.GetMapHistoryResponse{History: history, Forgetting: string(forgotten)}), nil
}
