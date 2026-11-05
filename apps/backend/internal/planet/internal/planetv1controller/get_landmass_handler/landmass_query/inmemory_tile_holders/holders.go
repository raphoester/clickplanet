package inmemory_tile_holders

import (
	"cmp"
	"slices"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Tiles interface {
	HoldingsOf(landmass clicks.LandmassID) []clicks.Holding
}

func New(tiles Tiles) Holders {
	return Holders{tiles: tiles}
}

type Holders struct {
	tiles Tiles
}

func (h Holders) Holders(landmass clicks.LandmassID) []*planetv1.LandmassHolder {
	holdings := h.tiles.HoldingsOf(landmass)

	holders := make([]*planetv1.LandmassHolder, 0, len(holdings))
	for _, holding := range holdings {
		holders = append(holders, &planetv1.LandmassHolder{
			AccountId: holding.Account.String(),
			CountryId: holding.Flag,
			Tiles:     uint32(holding.Tiles), //nolint:gosec // at most the tiles of one landmass.
		})
	}
	slices.SortFunc(holders, func(a, b *planetv1.LandmassHolder) int {
		return cmp.Or(
			cmp.Compare(b.GetTiles(), a.GetTiles()),
			cmp.Compare(a.GetAccountId(), b.GetAccountId()),
			cmp.Compare(a.GetCountryId(), b.GetCountryId()),
		)
	})

	return holders
}
