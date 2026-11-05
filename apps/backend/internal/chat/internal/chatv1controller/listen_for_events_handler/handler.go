package listen_for_events_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type UseCase interface {
	Execute(ctx context.Context, in listen_for_events_usecase.In, sink listen_for_events_usecase.Sink) error
}

func New(useCase UseCase) ListenForEventsHandler {
	return ListenForEventsHandler{useCase: useCase}
}

type ListenForEventsHandler struct {
	useCase UseCase
}

func (h ListenForEventsHandler) ListenForEvents(
	ctx context.Context,
	req *connect.Request[chatv1.ListenForEventsRequest],
	stream *connect.ServerStream[chatv1.ChatEvent],
) error {
	err := h.useCase.Execute(ctx, listen_for_events_usecase.In{
		Viewer: messages.AccountIDOf(cpctx.GetAccount(ctx)),
		Room:   messages.RoomOf(req.Msg.GetFlagRoom()),
	}, NewSink(stream))

	switch {
	case errors.Is(err, messages.ErrNoAccount):
		return connect.NewError(connect.CodeUnauthenticated, messages.ErrNoAccount)
	case errors.Is(err, messages.ErrNotInRoom):
		return connect.NewError(connect.CodeFailedPrecondition, messages.ErrNotInRoom)
	default:
		return err
	}
}
