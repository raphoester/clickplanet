package listen_for_events_handler

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/claim_bonus_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/planetmessage"
)

type EventStream interface {
	Send(event *planetv1.PlanetEvent) error
}

func NewSink(stream EventStream) Sink {
	return Sink{stream: stream}
}

type Sink struct {
	stream EventStream
}

var _ listen_for_events_usecase.Sink = Sink{}

func (s Sink) Send(event listen_for_events_usecase.Event) error {
	switch {
	case event.Heartbeat:
		return s.stream.Send(heartbeatEvent())
	case event.Offer != nil:
		return s.stream.Send(bonusOfferedEvent(event.Offer))
	case event.Quiz != nil:
		return s.stream.Send(quizOfferedEvent(event.Quiz))
	case event.Taken != nil:
		return s.stream.Send(bonusTakenEvent(event.Taken))
	case event.Blast != nil:
		return s.stream.Send(planetmessage.BombDropped(event.Blast))
	case event.Fortification != nil:
		return s.stream.Send(planetmessage.LandmassFortified(event.Fortification))
	case event.Enclosed != nil:
		enclosed := event.Enclosed
		return s.stream.Send(planetmessage.TilesEnclosed(
			enclosed.CountryID, enclosed.ClosingTile, enclosed.Wall, enclosed.Filled, enclosed.Yours))
	case event.Spread != nil:
		return s.stream.Send(planetmessage.TilesSpread(event.Spread.CountryID, event.Spread.Tile, event.Spread.Neighbours))
	default:
		return s.stream.Send(planetmessage.TileUpdate(event.Update))
	}
}

func bonusOfferedEvent(offer *bonuses.Offer) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_BonusOffered{
			BonusOffered: &planetv1.BonusOffered{
				Token:           offer.Token,
				Seed:            offer.Seed,
				Kind:            claim_bonus_handler.EncodeKind(offer.Kind),
				ExpiresAtUnixMs: offer.ExpiresAt.UnixMilli(),
			},
		},
	}
}

// Token and expiry only: anything about the question here can give its answer away.
func quizOfferedEvent(offer *bonuses.QuizOffer) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_QuizOffered{
			QuizOffered: &planetv1.QuizOffered{
				Token:           offer.Token,
				ExpiresAtUnixMs: offer.ExpiresAt.UnixMilli(),
			},
		},
	}
}

func bonusTakenEvent(taken *bonuses.Taken) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_BonusTaken{
			BonusTaken: &planetv1.BonusTaken{
				CountryId:            taken.CountryID,
				Kind:                 claim_bonus_handler.EncodeKind(taken.Kind),
				QuizSubjectCountryId: taken.QuizSubject,
			},
		},
	}
}

func heartbeatEvent() *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_Heartbeat{Heartbeat: &planetv1.Heartbeat{}},
	}
}
