package listen_for_events_handler

import (
	"context"

	"connectrpc.com/connect"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/listen_for_warnings_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, account string, sink listen_for_warnings_usecase.Sink) error
}

func New(warnings UseCase) ListenForEventsHandler {
	return ListenForEventsHandler{warnings: warnings}
}

type ListenForEventsHandler struct {
	warnings UseCase
}

func (h ListenForEventsHandler) ListenForEvents(
	ctx context.Context,
	_ *connect.Request[antibotv1.ListenForEventsRequest],
	stream *connect.ServerStream[antibotv1.AntibotEvent],
) error {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return err //nolint:wrapcheck // already the connect error the caller reads.
	}
	return h.warnings.Execute(ctx, account, NewSink(stream)) //nolint:wrapcheck // the stream's own error.
}
