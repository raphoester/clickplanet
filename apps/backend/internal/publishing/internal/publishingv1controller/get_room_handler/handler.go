package get_room_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/publishingwire"
)

type Query interface {
	Room(ctx context.Context, platforms []calendar.Platform) ([]*publishingv1.PlatformRoom, error)
}

func New(query Query) GetRoomHandler {
	return GetRoomHandler{query: query}
}

type GetRoomHandler struct {
	query Query
}

func (h GetRoomHandler) GetRoom(
	ctx context.Context,
	req *connect.Request[publishingv1.GetRoomRequest],
) (*connect.Response[publishingv1.GetRoomResponse], error) {
	platforms, err := publishingwire.Platforms(req.Msg.GetPlatforms())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	rooms, err := h.query.Room(ctx, platforms)
	var busy *posts.BusyError
	switch {
	case errors.As(err, &busy):
		return nil, connect.NewError(connect.CodeUnavailable, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&publishingv1.GetRoomResponse{Rooms: rooms}), nil
}
