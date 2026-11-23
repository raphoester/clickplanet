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

	Shields(tile uint32) int
	Shield(ctx context.Context, tile uint32, country string, most int, placement PlacementID) (int, error)
	Claim(ctx context.Context, write func(tiles Claimable) error) error
	Enclose(ctx context.Context, enclosure Enclosure, write func(tiles Claimable) error) error
	Infect(ctx context.Context, write func(tiles Infecting) error) error
	// Every live virus of the map in play.
	Viruses() []Virus
	Lay(ctx context.Context, write func(tiles Laying) error) (MapVersion, error)
	Explode(ctx context.Context, write func(tiles Exploding) error) error
	// The tiles the flag's mines are on, and the map's version as they were read.
	MinesOf(flag string) ([]uint32, MapVersion)
	// The tiles the flag's traps are on, and the map's version as they were read.
	TrapsOf(flag string) ([]uint32, MapVersion)
	// A value each time a take sets a mine off: the explosion waits for Explode.
	Fuse() <-chan struct{}
	Fortresses() map[ProvinceID]string
	// A player's take holds its tile for the account in its context; any other write leaves the tile held by nobody.
	HoldingsOf(landmass LandmassID) []Holding
	ForgetHolder(ctx context.Context, account cpsession.AccountID) (int, error)
	Clear(ctx context.Context, blast Blast) (Blast, error)
	Restore(ctx context.Context, restorations []Restoration) (int, error)

	// The version is the map's as the subscription starts: every change after it is on the channel, in order.
	Subscribe(ctx context.Context) (<-chan Change, MapVersion, error)
}
