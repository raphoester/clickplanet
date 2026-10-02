package listen_for_events_handler

import (
	"context"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events_usecase"
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

func (h ListenForEventsHandler) ListenForEvents(
	ctx context.Context,
	_ *connect.Request[planetv1.ListenForEventsRequest],
	stream *connect.ServerStream[planetv1.PlanetEvent],
) error {
	return h.useCase.Execute(ctx, NewSink(stream))
}
