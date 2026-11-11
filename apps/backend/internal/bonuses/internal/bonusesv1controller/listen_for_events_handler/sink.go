package listen_for_events_handler

import (
	"connectrpc.com/connect"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/planetwords"
)

type Stream interface {
	Send(event *bonusesv1.BonusEvent) error
}

var _ Stream = (*connect.ServerStream[bonusesv1.BonusEvent])(nil)

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

func frameOf(event listen_for_events_usecase.Event) *bonusesv1.BonusEvent {
	switch {
	case event.Heartbeat:
		return &bonusesv1.BonusEvent{Event: &bonusesv1.BonusEvent_Heartbeat{Heartbeat: &bonusesv1.Heartbeat{}}}
	case event.RulesChanged:
		return &bonusesv1.BonusEvent{Event: &bonusesv1.BonusEvent_RulesChanged{RulesChanged: &bonusesv1.RulesChanged{}}}
	case event.Offer != nil:
		return &bonusesv1.BonusEvent{Event: &bonusesv1.BonusEvent_BonusOffered{BonusOffered: &bonusesv1.BonusOffered{
			Token:           event.Offer.Token,
			Seed:            event.Offer.Seed,
			Kind:            planetwords.Kind(event.Offer.Kind),
			ExpiresAtUnixMs: event.Offer.ExpiresAt.UnixMilli(),
		}}}
	case event.Quiz != nil:
		return &bonusesv1.BonusEvent{Event: &bonusesv1.BonusEvent_QuizOffered{QuizOffered: &bonusesv1.QuizOffered{
			Token:           event.Quiz.Token,
			ExpiresAtUnixMs: event.Quiz.ExpiresAt.UnixMilli(),
		}}}
	case event.Taken != nil:
		return &bonusesv1.BonusEvent{Event: &bonusesv1.BonusEvent_BonusTaken{BonusTaken: &bonusesv1.BonusTaken{
			CountryId:            event.Taken.CountryID,
			Kind:                 planetwords.Kind(event.Taken.Kind),
			QuizSubjectCountryId: event.Taken.QuizSubject,
		}}}
	}
	return nil
}
