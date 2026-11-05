package clicks

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type TileStorage interface {
	Owner(tile uint32) (string, bool)
	Share(country string) float64
	Held(country string) int
	StateBatchDense(start uint32, end uint32) (DenseBatch, error)

	Set(ctx context.Context, tile uint32, value string) error
	Click(ctx context.Context, tile uint32, value string) error
	Shields(tile uint32) int
	Strike(ctx context.Context, tile uint32, owner string) bool
	Shield(ctx context.Context, tile uint32, country string, most int) error
	Fortify(ctx context.Context, tile uint32, flag string, most int) (Fortification, error)
	Fortresses() map[LandmassID]string
	// A player's take holds its tile for the account in its context; any other write leaves the tile held by nobody.
	HoldingsOf(landmass LandmassID) []Holding
	ForgetHolder(ctx context.Context, account cpsession.AccountID) (int, error)
	Clear(ctx context.Context, blast Blast) (Blast, error)
	Reassign(ctx context.Context, from, to string, start uint32, limit int) (next uint32, moved int, err error)
	Restore(ctx context.Context, restorations []Restoration) (int, error)

	// The version is the map's as the subscription starts: every change after it is on the channel, in order.
	Subscribe(ctx context.Context) (<-chan Change, MapVersion, error)
}
