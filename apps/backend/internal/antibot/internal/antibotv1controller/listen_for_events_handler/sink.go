package listen_for_events_handler

import (
	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/listen_for_warnings_usecase"
)

type EventStream interface {
	Send(event *antibotv1.AntibotEvent) error
}

func NewSink(stream EventStream) *Sink {
	return &Sink{stream: stream}
}

type Sink struct {
	stream EventStream
}

var _ listen_for_warnings_usecase.Sink = (*Sink)(nil)

func (s *Sink) SendHeartbeat() error {
	return s.stream.Send(&antibotv1.AntibotEvent{ //nolint:wrapcheck // the stream's own error.
		Event: &antibotv1.AntibotEvent_Heartbeat{Heartbeat: &antibotv1.Heartbeat{}},
	})
}

func (s *Sink) SendWarning() error {
	return s.stream.Send(&antibotv1.AntibotEvent{ //nolint:wrapcheck // the stream's own error.
		Event: &antibotv1.AntibotEvent_Warning{Warning: &antibotv1.Warning{}},
	})
}
