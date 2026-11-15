package listen_for_events_handler

import (
	"connectrpc.com/connect"

	boxesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes/usecases/listen_for_events_usecase"
)

type Stream interface {
	Send(event *boxesv1.BoxEvent) error
}

var _ Stream = (*connect.ServerStream[boxesv1.BoxEvent])(nil)

func NewSink(stream Stream) Sink {
	return Sink{stream: stream}
}

type Sink struct {
	stream Stream
}

func (s Sink) Send(event listen_for_events_usecase.Event) error {
	if frame := frameOf(event); frame != nil {
		return s.stream.Send(frame)
	}
	return nil
}

func frameOf(event listen_for_events_usecase.Event) *boxesv1.BoxEvent {
	switch {
	case event.Heartbeat:
		return &boxesv1.BoxEvent{Event: &boxesv1.BoxEvent_Heartbeat{Heartbeat: &boxesv1.Heartbeat{}}}
	case event.Offer != nil:
		return &boxesv1.BoxEvent{Event: &boxesv1.BoxEvent_BoxOffered{BoxOffered: &boxesv1.BoxOffered{
			Token:           event.Offer.Token,
			Seed:            event.Offer.Seed,
			ExpiresAtUnixMs: event.Offer.ExpiresAt.UnixMilli(),
		}}}
	}
	return nil
}
