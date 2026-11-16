package listen_for_events_handler

import (
	"errors"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/chargesheld"
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

var errEmptyEvent = errors.New("an event with nothing to send")

func (s Sink) Send(event listen_for_events_usecase.Event) error {
	switch {
	case event.Heartbeat:
		return s.stream.Send(heartbeatEvent(event.Version))
	case event.RulesChanged:
		return s.stream.Send(&planetv1.PlanetEvent{
			Event: &planetv1.PlanetEvent_RulesChanged{RulesChanged: &planetv1.RulesChanged{}},
		})
	case event.Update != nil:
		return s.stream.Send(planetmessage.TileUpdate(*event.Update, event.Version))
	case event.Blast != nil:
		return s.stream.Send(planetmessage.BombDropped(event.Blast, event.Version))
	case event.Fortification != nil:
		return s.stream.Send(planetmessage.LandmassFortified(event.Fortification, event.Version))
	case event.KnockOut != nil:
		return s.stream.Send(planetmessage.KnockOut(event.KnockOut, event.Version))
	case event.Tally != nil:
		return s.stream.Send(planetmessage.HoldingsChanged(event.Tally, event.Version))
	case event.Infection != nil:
		return s.stream.Send(planetmessage.VirusSpread(event.Infection, event.Version))
	case event.Explosion != nil:
		return s.stream.Send(planetmessage.MinesExploded(event.Explosion, event.Version))
	case event.Rally != nil:
		return s.stream.Send(planetmessage.RallyCalled(event.Rally.Rally, event.Rally.Left))
	case event.MineLaid != nil:
		return s.stream.Send(planetmessage.MineLaid(event.MineLaid.Flag, event.MineLaid.Tile, event.MineLaid.Version))
	case event.Enclosure != nil:
		return s.stream.Send(planetmessage.TilesEnclosed(event.Enclosure, event.Version, event.Yours))
	case event.Spread != nil:
		return s.stream.Send(planetmessage.TilesSpread(event.Spread.CountryID, event.Spread.Tile, event.Spread.Neighbours))
	case event.Granted != nil:
		return s.stream.Send(&planetv1.PlanetEvent{
			Event: &planetv1.PlanetEvent_ChargesGranted{ChargesGranted: &planetv1.ChargesGranted{
				Charges: chargesheld.Encode(event.Granted.After),
				Gained:  chargesheld.Gained(*event.Granted),
			}},
		})
	default:
		return errEmptyEvent
	}
}

func heartbeatEvent(version clicks.MapVersion) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_Heartbeat{Heartbeat: &planetv1.Heartbeat{Version: planetmessage.Version(version)}},
	}
}
