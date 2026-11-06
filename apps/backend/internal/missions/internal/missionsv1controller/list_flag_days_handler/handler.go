package list_flag_days_handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/list_flag_days_handler/flag_days_query"
)

type Query interface {
	FlagDays(ctx context.Context, since, until time.Time) (*missionsv1.ListFlagDaysResponse, error)
}

func New(query Query) ListFlagDaysHandler {
	return ListFlagDaysHandler{query: query}
}

type ListFlagDaysHandler struct {
	query Query
}

func (h ListFlagDaysHandler) ListFlagDays(
	ctx context.Context,
	req *connect.Request[missionsv1.ListFlagDaysRequest],
) (*connect.Response[missionsv1.ListFlagDaysResponse], error) {
	var since, until time.Time
	if at := req.Msg.GetSince(); at != nil {
		since = at.AsTime()
	}
	if at := req.Msg.GetUntil(); at != nil {
		until = at.AsTime()
	}

	answer, err := h.query.FlagDays(ctx, since, until)
	if errors.Is(err, flag_days_query.ErrInvalidWindow) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	return connect.NewResponse(answer), nil
}
