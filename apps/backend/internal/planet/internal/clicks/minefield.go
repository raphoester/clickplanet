package clicks

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

// Laying is the tiles under the lock of a mine laid: nothing else reads or writes them until it ends.
type Laying interface {
	Tile(id uint32) (Tile, error)
	Mined(tile uint32) bool
	Put(mine Mine)
}

// Exploding is the tiles under the lock of the mines a take set off: nothing else reads or writes them until it ends.
type Exploding interface {
	Claimable
	// TakeTripped takes off the map the next mine a take set off; false when none is left.
	TakeTripped() (Mine, bool)
	// SetOff takes off the map the mine on the tile, which a blast reached; false for a tile with none.
	SetOff(tile uint32) (Mine, bool)
	// Blasted keeps a mine gone off for the one change the explosion is.
	Blasted(blast MineBlast)
}

type MinableTiles interface {
	// Lay holds the lock through `write`, and answers the map's version under it: a lay changes no tile.
	Lay(ctx context.Context, write func(tiles Laying) error) (MapVersion, error)
	// Explode holds the lock through every write of `write`: one change at one version with every blast, then the
	// fortifies they earned, then the knock-outs.
	Explode(ctx context.Context, write func(tiles Exploding) error) error
}

func NewMinefield(tiles MinableTiles, ground Neighbourhood, shieldsPerTile func() int) Minefield {
	return Minefield{tiles: tiles, ground: ground, shieldsPerTile: shieldsPerTile}
}

// Minefield plays the mines on the map: one laid, and the ones a take set off, each under one lock of the tiles.
type Minefield struct {
	tiles          MinableTiles
	ground         Neighbourhood
	shieldsPerTile func() int
}

// Lay puts the mine on its tile, which must be its flag's and hold no mine, and answers the map's version.
func (f Minefield) Lay(ctx context.Context, mine Mine) (MapVersion, error) {
	return f.tiles.Lay(ctx, func(tiles Laying) error { //nolint:wrapcheck // a pure delegation: the storage names it.
		tile, err := tiles.Tile(mine.tile)
		if err != nil {
			return err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
		}
		laid, err := mine.Laid(tile, tiles.Mined(mine.tile))
		if err != nil {
			return err
		}
		tiles.Put(laid)

		return nil
	})
}

// Detonate sets off every mine a take tripped. Each blast takes the tiles within rings of its own for its mine's flag,
// nearest first, and every mine it reaches goes off after it, whatever its flag. A capital a blast took knocks its flag
// out once every blast is written. A failure stops the explosion, and what it wrote before is kept and told all the same.
func (f Minefield) Detonate(ctx context.Context, rings int) ([]Detonation, error) {
	most := f.shieldsPerTile()
	var detonations []Detonation
	err := f.tiles.Explode(ctx, func(tiles Exploding) error {
		var waiting []Mine
		for {
			if len(waiting) == 0 {
				tripped, ok := tiles.TakeTripped()
				if !ok {
					break
				}
				waiting = append(waiting, tripped)
			}
			mine := waiting[0]
			waiting = waiting[1:]

			detonation, reached, err := f.blast(ctx, tiles, mine, rings, most)
			if err != nil {
				return err
			}
			detonations = append(detonations, detonation)
			waiting = append(waiting, reached...)
		}

		for i := range detonations {
			ctx := cpctx.AddAccountToContext(ctx, detonations[i].Mine.Account.String())
			impacts, err := knockedOutAll(ctx, tiles, detonations[i].Mine.Flag, detonations[i].Impacts)
			detonations[i].Impacts = impacts
			if err != nil {
				return err
			}
		}

		return nil
	})

	return detonations, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
}

// blast is one mine gone off: every tile within rings of its own, its own first, and the mines it reached.
func (f Minefield) blast(ctx context.Context, tiles Exploding, mine Mine, rings, most int) (Detonation, []Mine, error) {
	ctx = cpctx.AddAccountToContext(ctx, mine.account.String())
	var (
		reached []Mine
		impacts []Impact
		taken   []uint32
	)
	for _, id := range append([]uint32{mine.tile}, Disc(f.ground, mine.tile, rings)...) {
		if other, ok := tiles.SetOff(id); ok {
			reached = append(reached, other)
		}
		tile, err := tiles.Tile(id)
		if err != nil {
			return Detonation{}, nil, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
		}
		blasted, impact := mine.blasted(tile)
		if impact.Outcome != Taken {
			continue
		}
		if err := tiles.Set(ctx, blasted); err != nil {
			return Detonation{}, nil, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
		}
		impact, err = fortifiedAfter(ctx, tiles, impact, mine.flag, most)
		if err != nil {
			return Detonation{}, nil, err
		}
		impacts = append(impacts, impact)
		taken = append(taken, id)
	}

	kept := mine.TakeSnapshot()
	tiles.Blasted(MineBlast{Mine: kept, Taken: taken})

	return Detonation{Mine: kept, Impacts: impacts}, reached, nil
}
