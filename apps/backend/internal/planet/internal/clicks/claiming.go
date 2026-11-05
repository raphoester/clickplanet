package clicks

import (
	"context"
	"errors"
)

type Impact struct {
	Tile    uint32
	Owner   string
	Outcome Outcome
	Shields int
	// Set when this take made the tile's landmass whole and fortified it.
	Fortified *Fortification
}

type Claimable interface {
	Shields
	Owner(tile uint32) (string, bool)
	Set(ctx context.Context, tile uint32, value string) error
	Click(ctx context.Context, tile uint32, value string) error
	Fortify(ctx context.Context, tile uint32, flag string, most int) (Fortification, error)
}

type Enclosing interface {
	Enclose(ctx context.Context, enclosure Enclosure, write func(tiles Claimable) error) error
}

type ClaimableTiles interface {
	Claimable
	Enclosing
}

func NewClaiming(tiles ClaimableTiles, shieldsPerTile func() int) Claiming {
	return Claiming{tiles: tiles, shieldsPerTile: shieldsPerTile}
}

type Claiming struct {
	tiles          ClaimableTiles
	shieldsPerTile func() int
}

func (c Claiming) Click(ctx context.Context, tile uint32, flag string) (Impact, error) {
	return claim(ctx, c.tiles, tile, flag, c.tiles.Click, c.shieldsPerTile())
}

func (c Claiming) Claim(ctx context.Context, tile uint32, flag string) (Impact, error) {
	return claim(ctx, c.tiles, tile, flag, c.tiles.Set, c.shieldsPerTile())
}

// Enclose claims the tiles inside as one write, which every stream hears as one change.
func (c Claiming) Enclose(ctx context.Context, enclosure Enclosure, inside []uint32) ([]Impact, error) {
	enclosure.By = PayerOf(ctx)
	most := c.shieldsPerTile()
	impacts := make([]Impact, 0, len(inside))
	err := c.tiles.Enclose(ctx, enclosure, func(tiles Claimable) error {
		for _, tile := range inside {
			impact, err := claim(ctx, tiles, tile, enclosure.Country, tiles.Set, most)
			if err != nil {
				return err
			}
			impacts = append(impacts, impact)
		}

		return nil
	})

	return impacts, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
}

func claim(
	ctx context.Context, tiles Claimable, tile uint32, flag string, write func(context.Context, uint32, string) error, most int,
) (Impact, error) {
	owner, _ := tiles.Owner(tile)
	outcome := NewShielding(tiles).Strike(ctx, tile, owner, flag)

	if err := write(ctx, tile, outcome.OwnerAfter(owner, flag)); err != nil {
		return Impact{}, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
	}

	var fortified *Fortification
	if outcome == Taken {
		fortification, err := tiles.Fortify(ctx, tile, flag, most)
		switch {
		case err == nil:
			fortified = &fortification
		case !errors.Is(err, ErrNotWhole) && !errors.Is(err, ErrFortifiedAlready):
			return Impact{}, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
		}
	}

	return Impact{Tile: tile, Owner: owner, Outcome: outcome, Shields: tiles.Shields(tile), Fortified: fortified}, nil
}
