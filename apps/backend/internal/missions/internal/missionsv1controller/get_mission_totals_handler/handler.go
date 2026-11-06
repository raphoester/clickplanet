package get_mission_totals_handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/get_mission_totals_handler/totals_query"
)

type Query interface {
	Totals(ctx context.Context, since, until time.Time) (*missionsv1.GetMissionTotalsResponse, error)
}

func New(query Query) GetMissionTotalsHandler {
	return GetMissionTotalsHandler{query: query}
}

type GetMissionTotalsHandler struct {
	query Query
}

func (h GetMissionTotalsHandler) GetMissionTotals(
	ctx context.Context,
	req *connect.Request[missionsv1.GetMissionTotalsRequest],
) (*connect.Response[missionsv1.GetMissionTotalsResponse], error) {
	var since, until time.Time
	if at := req.Msg.GetSince(); at != nil {
		since = at.AsTime()
	}
	if at := req.Msg.GetUntil(); at != nil {
		until = at.AsTime()
	}

	answer, err := h.query.Totals(ctx, since, until)
	if errors.Is(err, totals_query.ErrInvalidWindow) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	return connect.NewResponse(answer), nil
}
