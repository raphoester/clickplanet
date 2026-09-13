package listen_for_events_handler

import (
	"time"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/claim_bonus_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/bonus"
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
	switch {
	case event.Heartbeat:
		return s.stream.Send(heartbeatEvent())
	case event.Offer != nil:
		return s.stream.Send(bonusOfferedEvent(event.Offer))
	case event.Taken != nil:
		return s.stream.Send(bonusTakenEvent(event.Taken))
	case event.Blast != nil:
		return s.stream.Send(bombDroppedEvent(event.Blast))
	case event.Enclosed != nil:
		return s.stream.Send(tilesEnclosedEvent(event.Enclosed))
	case event.Spread != nil:
		return s.stream.Send(tilesSpreadEvent(event.Spread))
	case event.Boosted != nil:
		return s.stream.Send(clickBoostedEvent(event.Boosted))
	default:
		return s.stream.Send(tileUpdateEvent(event.Update))
	}
}

func bonusOfferedEvent(offer *bonus.Offer) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_BonusOffered{
			BonusOffered: &planetv1.BonusOffered{
				Token:           offer.Token,
				Seed:            offer.Seed,
				Kind:            claim_bonus_handler.EncodeKind(offer.Kind),
				DurationSeconds: uint32(offer.Duration / time.Second),
				ExpiresAtUnixMs: offer.ExpiresAt.UnixMilli(),
			},
		},
	}
}

func bonusTakenEvent(taken *bonus.Taken) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_BonusTaken{
			BonusTaken: &planetv1.BonusTaken{
				CountryId: taken.CountryID,
				Kind:      claim_bonus_handler.EncodeKind(taken.Kind),
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

func tilesEnclosedEvent(enclosed *bonus.Enclosed) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_TilesEnclosed{
			TilesEnclosed: &planetv1.TilesEnclosed{
				CountryId:      enclosed.CountryID,
				ClosingTileId:  enclosed.ClosingTile,
				WallTileIds:    enclosed.Wall,
				FilledTileIds:  enclosed.Filled,
				Yours:          enclosed.Yours,
				EnclosuresLeft: uint32(enclosed.Left),
			},
		},
	}
}

func tilesSpreadEvent(spread *bonus.Spread) *planetv1.PlanetEvent {
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

func clickBoostedEvent(boosted *bonus.Boosted) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_ClickBoosted{
			ClickBoosted: &planetv1.ClickBoosted{
				CountryId: boosted.CountryID,
				TileId:    boosted.Tile,
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
