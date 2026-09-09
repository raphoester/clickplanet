package planetv1controller

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain"
	"google.golang.org/protobuf/proto"
)

// TileUpdateRoute is where the tile stream is served, under the app's /ws
// prefix.
const TileUpdateRoute = "/listen"

// EncodeTileUpdate is what the websocket fanout sends: a bare TileUpdate with
// no envelope around it, which is the frame shape every deployed client
// already reads.
func EncodeTileUpdate(update domain.TileUpdate) ([]byte, error) {
	return proto.Marshal(&planetv1.TileUpdate{
		TileId:            update.Tile,
		CountryId:         update.Value,
		PreviousCountryId: update.Previous,
	})
}
