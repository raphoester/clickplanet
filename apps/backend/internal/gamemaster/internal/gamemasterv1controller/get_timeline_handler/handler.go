package get_timeline_handler

import (
	"context"

	"connectrpc.com/connect"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
)

// The turn's read: the timeline's battles are the ones the players see coming, and a module reads the same.
type Query interface {
	Now(ctx context.Context) (*gamemasterv1.GetTurnResponse, error)
}

func New(query Query) GetTimelineHandler {
	return GetTimelineHandler{query: query}
}

type GetTimelineHandler struct {
	query Query
}

func (h GetTimelineHandler) GetTimeline(
	ctx context.Context,
	_ *connect.Request[gamemasterv1.GetTimelineRequest],
) (*connect.Response[gamemasterv1.GetTimelineResponse], error) {
	now, err := h.query.Now(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&gamemasterv1.GetTimelineResponse{Battles: now.GetTimeline().GetBattles()}), nil
}
