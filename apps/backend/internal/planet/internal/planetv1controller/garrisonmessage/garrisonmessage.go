package garrisonmessage

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/garrisons"
)

func Encode(garrison garrisons.Garrison) *planetv1.Garrison {
	return &planetv1.Garrison{
		TileId:    garrison.Tile,
		CountryId: garrison.Country,
		Defenders: uint32(garrison.Defenders),
	}
}
