package get_schedule_handler

import (
	"context"

	"connectrpc.com/connect"

	rulesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/rules/v1"
)

type Query interface {
	Schedule(ctx context.Context) (*rulesv1.GetScheduleResponse, error)
}

func New(query Query) GetScheduleHandler {
	return GetScheduleHandler{query: query}
}

type GetScheduleHandler struct {
	query Query
}

func (h GetScheduleHandler) GetSchedule(
	ctx context.Context,
	_ *connect.Request[rulesv1.GetScheduleRequest],
) (*connect.Response[rulesv1.GetScheduleResponse], error) {
	answer, err := h.query.Schedule(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // a storage failure is the error net's to answer.
	}
	return connect.NewResponse(answer), nil
}
