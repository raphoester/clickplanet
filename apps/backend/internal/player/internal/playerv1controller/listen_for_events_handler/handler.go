// Package listen_for_events_handler serves player.v1.PlayerService/ListenForEvents.
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

func New(useCase UseCase) ListenForEventsHandler {
	return ListenForEventsHandler{useCase: useCase}
}

type ListenForEventsHandler struct {
	useCase UseCase
}

// ListenForEvents needs no token: the session interceptor does not list it.
func (h ListenForEventsHandler) ListenForEvents(
	ctx context.Context,
	_ *connect.Request[playerv1.ListenForEventsRequest],
	stream *connect.ServerStream[playerv1.PlayerEvent],
) error {
	return h.useCase.Execute(ctx, NewSink(stream))
}
