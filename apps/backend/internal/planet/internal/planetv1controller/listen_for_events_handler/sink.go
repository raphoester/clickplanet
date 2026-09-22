package listen_for_events_handler

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/claim_bonus_handler"
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
		return s.stream.Send(bombDroppedEvent(event.Blast))
	case event.Enclosed != nil:
		return s.stream.Send(tilesEnclosedEvent(event.Enclosed))
	case event.Spread != nil:
		return s.stream.Send(tilesSpreadEvent(event.Spread))
	default:
		return s.stream.Send(tileUpdateEvent(event.Update))
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

// The token and the clock, and nothing else: the question, its choices and even what it is about
// are read with OpenQuiz, which is what starts the five seconds. A stream that carried any of them
// would be a stream a client could read at leisure.
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

func bombDroppedEvent(blast *clicks.Blast) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_BombDropped{
			BombDropped: &planetv1.BombDropped{
				TileId:         blast.Tile,
				CountryId:      blast.CountryID,
				Radius:         blast.Radius,
				ClearedTileIds: blast.Cleared,
				Point:          &planetv1.GlobePoint{X: blast.Point.X, Y: blast.Point.Y, Z: blast.Point.Z},
			},
		},
	}
}

func tilesEnclosedEvent(enclosed *bonuses.Enclosed) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_TilesEnclosed{
			TilesEnclosed: &planetv1.TilesEnclosed{
				CountryId:     enclosed.CountryID,
				ClosingTileId: enclosed.ClosingTile,
				WallTileIds:   enclosed.Wall,
				FilledTileIds: enclosed.Filled,
				Yours:         enclosed.Yours,
			},
		},
	}
}

func tilesSpreadEvent(spread *bonuses.Spread) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_TilesSpread{
			TilesSpread: &planetv1.TilesSpread{
				CountryId:     spread.CountryID,
				TileId:        spread.Tile,
				SpreadTileIds: spread.Neighbours,
			},
		},
	}
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
