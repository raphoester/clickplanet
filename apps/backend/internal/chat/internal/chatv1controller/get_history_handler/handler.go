package get_history_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type Query interface {
	History(ctx context.Context, viewer messages.AccountID, room messages.Room) (*chatv1.GetHistoryResponse, error)
}

type Rooms interface {
	EntryError(ctx context.Context, account messages.AccountID, room messages.Room) error
}

func New(query Query, rooms Rooms) GetHistoryHandler {
	return GetHistoryHandler{query: query, rooms: rooms}
}

type GetHistoryHandler struct {
	query Query
	rooms Rooms
}

func (h GetHistoryHandler) GetHistory(
	ctx context.Context,
	req *connect.Request[chatv1.GetHistoryRequest],
) (*connect.Response[chatv1.GetHistoryResponse], error) {
	viewer := messages.AccountIDOf(cpctx.GetAccount(ctx))
	room := messages.RoomOf(req.Msg.GetFlagRoom())

	if err := h.rooms.EntryError(ctx, viewer, room); err != nil {
		return nil, toConnect(err)
	}

	history, err := h.query.History(ctx, viewer, room)
	if err != nil {
		return nil, err //nolint:wrapcheck // a storage failure is the error net's to answer.
	}

	res := connect.NewResponse(history)
	res.Header().Set("Cache-Control", "no-store")

	return res, nil
}

func toConnect(err error) error {
	switch {
	case errors.Is(err, messages.ErrNoAccount):
		return connect.NewError(connect.CodeUnauthenticated, messages.ErrNoAccount)
	case errors.Is(err, messages.ErrNotInRoom):
		return connect.NewError(connect.CodeFailedPrecondition, messages.ErrNotInRoom)
	default:
		return err //nolint:wrapcheck // a failure to read the flag is the error net's to answer.
	}
}
