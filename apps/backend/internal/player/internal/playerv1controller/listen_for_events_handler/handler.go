package listen_for_events_handler

import (
	"context"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/listen_for_events_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, sink listen_for_events_usecase.Sink) error
}

func New(roster UseCase) ListenForEventsHandler {
	return ListenForEventsHandler{roster: roster}
}

type ListenForEventsHandler struct {
	roster UseCase
}

func (h ListenForEventsHandler) ListenForEvents(
	ctx context.Context,
	_ *connect.Request[playerv1.ListenForEventsRequest],
	stream *connect.ServerStream[playerv1.PlayerEvent],
) error {
	return h.roster.Execute(ctx, NewSink(stream)) //nolint:wrapcheck // the stream's own error.
}
