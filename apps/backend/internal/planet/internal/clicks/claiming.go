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

func NewClaiming(tiles Claimable, most int) Claiming {
	return Claiming{tiles: tiles, shielding: NewShielding(tiles), most: most}
}

type Claiming struct {
	tiles     Claimable
	shielding Shielding
	most      int
}

func (c Claiming) Click(ctx context.Context, tile uint32, flag string) (Impact, error) {
	return c.claim(ctx, tile, flag, c.tiles.Click)
}

func (c Claiming) Claim(ctx context.Context, tile uint32, flag string) (Impact, error) {
	return c.claim(ctx, tile, flag, c.tiles.Set)
}

func (c Claiming) claim(ctx context.Context, tile uint32, flag string, write func(context.Context, uint32, string) error) (Impact, error) {
	owner, _ := c.tiles.Owner(tile)
	outcome := c.shielding.Strike(ctx, tile, owner, flag)

	if err := write(ctx, tile, outcome.OwnerAfter(owner, flag)); err != nil {
		return Impact{}, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
	}

	var fortified *Fortification
	if outcome == Taken {
		fortification, err := c.tiles.Fortify(ctx, tile, flag, c.most)
		switch {
		case err == nil:
			fortified = &fortification
		case !errors.Is(err, ErrNotWhole) && !errors.Is(err, ErrFortifiedAlready):
			return Impact{}, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
		}
	}

	return Impact{Tile: tile, Owner: owner, Outcome: outcome, Shields: c.tiles.Shields(tile), Fortified: fortified}, nil
}
