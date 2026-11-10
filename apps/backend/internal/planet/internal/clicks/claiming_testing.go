//go:build testing

package clicks

import "context"

// Seed puts flag on the tile with no claim's rule: it strikes no shield and fortifies nothing.
func Seed(ctx context.Context, tiles ClaimableTiles, id uint32, flag string) error {
	return seed(ctx, tiles, id, flag, Claimable.Set)
}

func SeedClick(ctx context.Context, tiles ClaimableTiles, id uint32, flag string) error {
	return seed(ctx, tiles, id, flag, Claimable.Click)
}

func Fortify(ctx context.Context, tiles ClaimableTiles, id uint32, flag string, most int) (Fortification, error) {
	var fortification Fortification
	err := tiles.Claim(ctx, func(locked Claimable) error {
		var err error
		fortification, err = locked.Fortify(ctx, id, flag, most)
		return err //nolint:wrapcheck // a test's seed.
	})

	return fortification, err //nolint:wrapcheck // a test's seed.
}

func seed(ctx context.Context, tiles ClaimableTiles, id uint32, flag string, write writing) error {
	return tiles.Claim(ctx, func(locked Claimable) error { //nolint:wrapcheck // a test's seed.
		tile, err := locked.Tile(id)
		if err != nil || tile.owner == flag {
			return err //nolint:wrapcheck // a test's seed.
		}
		return write(locked, ctx, Tile{id: id, owner: flag, holder: holderOf(ctx)})
	})
}
