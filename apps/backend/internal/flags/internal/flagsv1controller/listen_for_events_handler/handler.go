package listen_for_events_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/feed/usecases/listen_for_events_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, sink listen_for_events_usecase.Sink) error
}

func New(useCase UseCase) ListenForEventsHandler {
	return ListenForEventsHandler{useCase: useCase}
}

// ListenForEventsHandler reads no token: every page hears the same flags.
type ListenForEventsHandler struct {
	useCase UseCase
}

func (h ListenForEventsHandler) ListenForEvents(
	ctx context.Context,
	_ *connect.Request[flagsv1.ListenForEventsRequest],
	stream *connect.ServerStream[flagsv1.FlagEvent],
) error {
	return h.useCase.Execute(ctx, sink{stream: stream}) //nolint:wrapcheck // the stream's own error.
}

type sink struct {
	stream *connect.ServerStream[flagsv1.FlagEvent]
}

var errEmptyEvent = errors.New("an event with nothing to send")

func (s sink) Send(event listen_for_events_usecase.Event) error {
	switch {
	case event.Heartbeat:
		return s.stream.Send(&flagsv1.FlagEvent{ //nolint:wrapcheck // the stream's own error.
			Event: &flagsv1.FlagEvent_Heartbeat{Heartbeat: &flagsv1.Heartbeat{}},
		})
	case event.Flag != nil:
		return s.stream.Send(&flagsv1.FlagEvent{ //nolint:wrapcheck // the stream's own error.
			Event: &flagsv1.FlagEvent_FlagChanged{FlagChanged: flagOf(*event.Flag)},
		})
	}
	return errEmptyEvent
}

func flagOf(flag catalog.Snapshot) *flagsv1.Flag {
	return &flagsv1.Flag{Code: flag.ID.String(), Name: flag.Name.String(), Closed: flag.State == catalog.Closed}
}
