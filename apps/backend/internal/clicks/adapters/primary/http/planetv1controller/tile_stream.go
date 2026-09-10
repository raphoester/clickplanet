package planetv1controller

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain"
	"google.golang.org/protobuf/proto"
)

const TileUpdateRoute = "/listen"

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
