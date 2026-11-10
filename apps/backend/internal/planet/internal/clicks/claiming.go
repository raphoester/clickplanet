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
	// Set when this take felled a flag's capital in an arena.
	KnockedOut *KnockOut
}

// The tiles under the lock of one claim: nothing else reads or writes them until it ends.
type Claimable interface {
	Tile(id uint32) (Tile, error)
	Set(ctx context.Context, tile Tile) error
	Click(ctx context.Context, tile Tile) error
	Fortify(ctx context.Context, tile uint32, flag string, most int) (Fortification, error)
	KnockOut(ctx context.Context, tile uint32, flag string) (KnockOut, error)
}

type ClaimableTiles interface {
	Claim(ctx context.Context, write func(tiles Claimable) error) error
	Enclose(ctx context.Context, enclosure Enclosure, write func(tiles Claimable) error) error
}

func NewClaiming(tiles ClaimableTiles, shieldsPerTile func() int) Claiming {
	return Claiming{tiles: tiles, shieldsPerTile: shieldsPerTile}
}

type Claiming struct {
	tiles          ClaimableTiles
	shieldsPerTile func() int
}

func (c Claiming) Click(ctx context.Context, tile uint32, flag string) (Impact, error) {
	return c.claimOne(ctx, tile, flag, Claimable.Click)
}

func (c Claiming) Claim(ctx context.Context, tile uint32, flag string) (Impact, error) {
	return c.claimOne(ctx, tile, flag, Claimable.Set)
}

// Enclose claims the tiles inside as one write, which every stream hears as one change. A capital it takes knocks its flag
// out once the whole pocket is written, so the blank leaves the pocket's takes as they are.
func (c Claiming) Enclose(ctx context.Context, enclosure Enclosure, inside []uint32) ([]Impact, error) {
	enclosure.By = PayerOf(ctx)
	most := c.shieldsPerTile()
	impacts := make([]Impact, 0, len(inside))
	err := c.tiles.Enclose(ctx, enclosure, func(tiles Claimable) error {
		for _, tile := range inside {
			impact, err := claim(ctx, tiles, tile, enclosure.Country, Claimable.Set, most)
			if err != nil {
				return err
			}
			impacts = append(impacts, impact)
		}
		for i := range impacts {
			knocked, err := knockedOut(ctx, tiles, enclosure.Country, impacts[i])
			if err != nil {
				return err
			}
			impacts[i] = knocked
		}

		return nil
	})

	return impacts, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
}

func (c Claiming) claimOne(ctx context.Context, tile uint32, flag string, write writing) (Impact, error) {
	most := c.shieldsPerTile()
	var impact Impact
	err := c.tiles.Claim(ctx, func(tiles Claimable) error {
		taken, err := claim(ctx, tiles, tile, flag, write, most)
		if err != nil {
			return err
		}
		impact, err = knockedOut(ctx, tiles, flag, taken)
		return err
	})

	return impact, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
}

type writing func(tiles Claimable, ctx context.Context, tile Tile) error

func claim(ctx context.Context, tiles Claimable, id uint32, flag string, write writing, most int) (Impact, error) {
	tile, err := tiles.Tile(id)
	if err != nil {
		return Impact{}, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
	}

	claimed, impact := tile.Claimed(flag, holderOf(ctx))
	if err := write(tiles, ctx, claimed); err != nil {
		return Impact{}, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
	}
	if impact.Outcome != Taken {
		return impact, nil
	}

	fortification, err := tiles.Fortify(ctx, id, flag, most)
	switch {
	case err == nil:
		return impact.fortified(fortification), nil
	case errors.Is(err, ErrNotWhole), errors.Is(err, ErrFortifiedAlready):
		return impact, nil
	default:
		return Impact{}, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
	}
}

func knockedOut(ctx context.Context, tiles Claimable, flag string, impact Impact) (Impact, error) {
	if impact.Outcome != Taken {
		return impact, nil
	}

	knockOut, err := tiles.KnockOut(ctx, impact.Tile, flag)
	switch {
	case err == nil:
		impact.KnockedOut = &knockOut
		return impact, nil
	case errors.Is(err, ErrNoKnockOut):
		return impact, nil
	default:
		return Impact{}, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
	}
}

func (i Impact) fortified(fortification Fortification) Impact {
	i.Fortified = &fortification
	for _, raised := range fortification.Raised {
		if raised.Tile == i.Tile {
			i.Shields = raised.Shields
		}
	}

	return i
}
