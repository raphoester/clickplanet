// Package listen_for_events_handler serves planet.v1.ClickService/ListenForEvents.
package listen_for_events_handler

import (
	"context"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events"
)

type UseCase interface {
	Execute(ctx context.Context, sink listen_for_events.Sink) error
}

func New(useCase UseCase) ListenForEventsHandler {
	return ListenForEventsHandler{useCase: useCase}
}

type ListenForEventsHandler struct {
	useCase UseCase
}

// The request context is what unsubscribes, and it is cancelled however the
// stream ends. Everything this method could get wrong is in the sink, which is
// why the sink is a type of its own rather than a closure over the stream.
func (h ListenForEventsHandler) ListenForEvents(
	ctx context.Context,
	_ *connect.Request[planetv1.ListenForEventsRequest],
	stream *connect.ServerStream[planetv1.PlanetEvent],
) error {
	return h.useCase.Execute(ctx, NewSink(stream))
}
