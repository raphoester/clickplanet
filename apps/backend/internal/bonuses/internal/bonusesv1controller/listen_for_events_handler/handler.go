package listen_for_events_handler

import (
	"context"

	"connectrpc.com/connect"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/caller"
)

type UseCase interface {
	Execute(ctx context.Context, caller bonuses.Caller, sink listen_for_events_usecase.Sink) error
}

func New(useCase UseCase) ListenForEventsHandler {
	return ListenForEventsHandler{useCase: useCase}
}

type ListenForEventsHandler struct {
	useCase UseCase
}

// A stream with no token, or a bad one, attends as its address: the session reader refuses nothing.
func (h ListenForEventsHandler) ListenForEvents(
	ctx context.Context,
	_ *connect.Request[bonusesv1.ListenForEventsRequest],
	stream *connect.ServerStream[bonusesv1.BonusEvent],
) error {
	return h.useCase.Execute(ctx, caller.Of(ctx), NewSink(stream)) //nolint:wrapcheck // the stream's own error.
}
