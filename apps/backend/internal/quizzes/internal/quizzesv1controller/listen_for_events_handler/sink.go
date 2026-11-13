package listen_for_events_handler

import (
	"connectrpc.com/connect"

	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzes/usecases/listen_for_events_usecase"
)

type Stream interface {
	Send(event *quizzesv1.QuizEvent) error
}

var _ Stream = (*connect.ServerStream[quizzesv1.QuizEvent])(nil)

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

func frameOf(event listen_for_events_usecase.Event) *quizzesv1.QuizEvent {
	switch {
	case event.Heartbeat:
		return &quizzesv1.QuizEvent{Event: &quizzesv1.QuizEvent_Heartbeat{Heartbeat: &quizzesv1.Heartbeat{}}}
	case event.Offer != nil:
		return &quizzesv1.QuizEvent{Event: &quizzesv1.QuizEvent_QuizOffered{QuizOffered: &quizzesv1.QuizOffered{
			Token:           event.Offer.Token,
			ExpiresAtUnixMs: event.Offer.ExpiresAt.UnixMilli(),
		}}}
	case event.Won != nil:
		return &quizzesv1.QuizEvent{Event: &quizzesv1.QuizEvent_QuizWon{QuizWon: &quizzesv1.QuizWon{
			CountryId:        event.Won.CountryID,
			SubjectCountryId: event.Won.Subject,
		}}}
	}
	return nil
}
