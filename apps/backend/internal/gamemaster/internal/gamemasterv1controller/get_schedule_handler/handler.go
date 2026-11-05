package get_schedule_handler

import (
	"context"

	"connectrpc.com/connect"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
)

type Query interface {
	Schedule(ctx context.Context) (*gamemasterv1.GetScheduleResponse, error)
}

func New(query Query) GetScheduleHandler {
	return GetScheduleHandler{query: query}
}

type GetScheduleHandler struct {
	query Query
}

func (h GetScheduleHandler) GetSchedule(
	ctx context.Context,
	_ *connect.Request[gamemasterv1.GetScheduleRequest],
) (*connect.Response[gamemasterv1.GetScheduleResponse], error) {
	answer, err := h.query.Schedule(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // a storage failure is the error net's to answer.
	}
	return connect.NewResponse(answer), nil
}
