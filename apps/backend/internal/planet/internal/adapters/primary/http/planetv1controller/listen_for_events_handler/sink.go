package listen_for_events_handler

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events"
)

// EventStream is what a sink writes to. The generated server stream satisfies
// it; declaring the one method here rather than taking connect's concrete type
// is what lets the mapping below be tested without a live HTTP response.
type EventStream interface {
	Send(event *planetv1.PlanetEvent) error
}

func NewSink(stream EventStream) Sink {
	return Sink{stream: stream}
}

// Sink writes the use case's frames as the proto envelope. The oneof is the
// wire's business and stops here: the use case says update or heartbeat, and
// nothing about how either is framed.
type Sink struct {
	stream EventStream
}

var _ listen_for_events.Sink = Sink{}

func (s Sink) Send(event listen_for_events.Event) error {
	if event.Heartbeat {
		return s.stream.Send(heartbeatEvent())
	}

	return s.stream.Send(tileUpdateEvent(event.Update))
}

func toProto(update clicks.TileUpdate) *planetv1.TileUpdate {
	return &planetv1.TileUpdate{
		TileId:            update.Tile,
		CountryId:         update.Value,
		PreviousCountryId: update.Previous,
	}
}

func tileUpdateEvent(update clicks.TileUpdate) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_TileUpdate{TileUpdate: toProto(update)},
	}
}

func heartbeatEvent() *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_Heartbeat{Heartbeat: &planetv1.Heartbeat{}},
	}
}
