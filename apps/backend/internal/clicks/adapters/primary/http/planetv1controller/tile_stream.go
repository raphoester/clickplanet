package planetv1controller

import (
	"time"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain"
	"google.golang.org/protobuf/proto"
)

// Well under Cloudflare's ~125s idle cut, and cheap: a heartbeat is two bytes.
const DefaultHeartbeat = 30 * time.Second

const TileUpdateRoute = "/listen"

// The websocket frame stays a bare TileUpdate: already-deployed clients decode
// exactly this, and they never see a PlanetEvent.
func EncodeTileUpdate(update domain.TileUpdate) ([]byte, error) {
	return proto.Marshal(toProto(update))
}

func toProto(update domain.TileUpdate) *planetv1.TileUpdate {
	return &planetv1.TileUpdate{
		TileId:            update.Tile,
		CountryId:         update.Value,
		PreviousCountryId: update.Previous,
	}
}

func tileUpdateEvent(update domain.TileUpdate) *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_TileUpdate{TileUpdate: toProto(update)},
	}
}

func heartbeatEvent() *planetv1.PlanetEvent {
	return &planetv1.PlanetEvent{
		Event: &planetv1.PlanetEvent_Heartbeat{Heartbeat: &planetv1.Heartbeat{}},
	}
}
