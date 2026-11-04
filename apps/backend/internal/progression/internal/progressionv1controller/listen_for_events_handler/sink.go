package listen_for_events_handler

import (
	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/progressionmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles/usecases/listen_for_titles_usecase"
)

type EventStream interface {
	Send(event *progressionv1.ProgressionEvent) error
}

func NewSink(stream EventStream) *Sink {
	return &Sink{stream: stream}
}

type Sink struct {
	stream EventStream
}

var _ listen_for_titles_usecase.Sink = (*Sink)(nil)

func (s *Sink) SendHeartbeat() error {
	return s.stream.Send(&progressionv1.ProgressionEvent{
		Event: &progressionv1.ProgressionEvent_Heartbeat{Heartbeat: &progressionv1.Heartbeat{}},
	})
}

func (s *Sink) SendTitleEarned(standing titles.Standing) error {
	return s.stream.Send(&progressionv1.ProgressionEvent{
		Event: &progressionv1.ProgressionEvent_TitleEarned{TitleEarned: &progressionv1.TitleEarned{Title: progressionmessage.Title(standing)}},
	})
}
