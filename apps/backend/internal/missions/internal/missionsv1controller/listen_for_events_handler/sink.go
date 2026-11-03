package listen_for_events_handler

import (
	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/get_missions_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/missionmessage"
)

type EventStream interface {
	Send(event *missionsv1.MissionEvent) error
}

func NewSink(stream EventStream) Sink {
	return Sink{stream: stream}
}

type Sink struct {
	stream EventStream
}

var _ listen_for_events_usecase.Sink = Sink{}

func (s Sink) SendDay(day get_missions_usecase.Out) error {
	encoded, err := missionmessage.Day(day)
	if err != nil {
		return err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	return s.stream.Send(&missionsv1.MissionEvent{Event: &missionsv1.MissionEvent_Day{Day: encoded}})
}

func (s Sink) SendHeartbeat() error {
	return s.stream.Send(&missionsv1.MissionEvent{Event: &missionsv1.MissionEvent_Heartbeat{Heartbeat: &missionsv1.Heartbeat{}}})
}
