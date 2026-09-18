// Package listen_for_events_handler serves chat.v1.ChatService/ListenForEvents.
package listen_for_events_handler

import (
	"context"

	"connectrpc.com/connect"
	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed/usecases/listen_for_events_usecase"
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
	_ *connect.Request[chatv1.ListenForEventsRequest],
	stream *connect.ServerStream[chatv1.ChatEvent],
) error {
	return h.useCase.Execute(ctx, NewSink(stream))
}
