package listen_for_events_handler

import (
	"context"

	"connectrpc.com/connect"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles/usecases/listen_for_titles_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, account accounts.AccountID, sink listen_for_titles_usecase.Sink) error
}

func New(titles UseCase) ListenForEventsHandler {
	return ListenForEventsHandler{titles: titles}
}

type ListenForEventsHandler struct {
	titles UseCase
}

func (h ListenForEventsHandler) ListenForEvents(
	ctx context.Context,
	_ *connect.Request[progressionv1.ListenForEventsRequest],
	stream *connect.ServerStream[progressionv1.ProgressionEvent],
) error {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return err //nolint:wrapcheck // already the connect error the caller reads.
	}
	return h.titles.Execute(ctx, account, NewSink(stream)) //nolint:wrapcheck // the stream's own error.
}
