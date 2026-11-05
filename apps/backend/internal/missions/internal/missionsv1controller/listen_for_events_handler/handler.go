package listen_for_events_handler

import (
	"context"

	"connectrpc.com/connect"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/caller"
)

type UseCase interface {
	Execute(ctx context.Context, account missions.AccountID, sink listen_for_events_usecase.Sink) error
}

func New(useCase UseCase) ListenForEventsHandler {
	return ListenForEventsHandler{useCase: useCase}
}

type ListenForEventsHandler struct {
	useCase UseCase
}

func (h ListenForEventsHandler) ListenForEvents(
	ctx context.Context,
	_ *connect.Request[missionsv1.ListenForEventsRequest],
	stream *connect.ServerStream[missionsv1.MissionEvent],
) error {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return err //nolint:wrapcheck // already the connect error the caller reads.
	}

	return h.useCase.Execute(ctx, account, NewSink(stream)) //nolint:wrapcheck // the error net answers it.
}
