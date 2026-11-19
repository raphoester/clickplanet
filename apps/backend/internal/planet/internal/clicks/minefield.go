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

// Detonate sets off every mine a take tripped. Each blast claims the tiles within rings of its own for its mine's flag,
// nearest first, and every mine it reaches goes off after it, whatever its flag. Fortifies, then knock-outs, then
// cut-offs, wait for the last blast: a stream hears them after the one change. A failure stops it, and what it wrote is kept and told.
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

			detonation, reached, err := f.blast(ctx, tiles, mine, rings)
			if err != nil {
				return err
			}
			detonations = append(detonations, detonation)
			waiting = append(waiting, reached...)
		}

		for i := range detonations {
			ctx := cpctx.AddAccountToContext(ctx, detonations[i].Mine.Account.String())
			impacts, err := fortifiedAll(ctx, tiles, detonations[i].Mine.Flag, detonations[i].Impacts, most)
			detonations[i].Impacts = impacts
			if err != nil {
				return err
			}
		}
		for i := range detonations {
			ctx := cpctx.AddAccountToContext(ctx, detonations[i].Mine.Account.String())
			impacts, err := knockedOutAll(ctx, tiles, detonations[i].Mine.Flag, detonations[i].Impacts)
			detonations[i].Impacts = impacts
			if err != nil {
				return err
			}
		}

		var takes []take
		for i := range detonations {
			takes = append(takes, takesOf(detonations[i].Mine.Flag, detonations[i].Impacts)...)
		}
		return cutOffAll(tiles, takes)
	})

	return detonations, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
}

// blast is one mine gone off: every tile within rings of its own, its own first, and the mines it reached.
func (f Minefield) blast(ctx context.Context, tiles Exploding, mine Mine, rings int) (Detonation, []Mine, error) {
	ctx = cpctx.AddAccountToContext(ctx, mine.account.String())
	var (
		reached []Mine
		impacts []Impact
		taken   []uint32
		struck  []TileShields
	)
	for _, id := range append([]uint32{mine.tile}, Disc(f.ground, mine.tile, rings)...) {
		if other, ok := tiles.SetOff(id); ok {
			reached = append(reached, other)
		}
		tile, err := tiles.Tile(id)
		if err != nil {
			return Detonation{}, nil, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
		}
		blasted, impact := tile.Claimed(mine.flag, mine.account)
		switch impact.Outcome {
		case Unchanged:
			continue
		case Shielded:
			struck = append(struck, TileShields{Tile: id, Shields: impact.Shields})
		case Taken:
			taken = append(taken, id)
		}
		if err := tiles.Set(ctx, blasted); err != nil {
			return Detonation{}, nil, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
		}
		impacts = append(impacts, impact)
	}

	kept := mine.TakeSnapshot()
	tiles.Blasted(MineBlast{Mine: kept, Taken: taken, Struck: struck})

	return Detonation{Mine: kept, Impacts: impacts}, reached, nil
}

// fortifiedAll asks each take whether it made its province whole.
func fortifiedAll(ctx context.Context, tiles Claimable, flag string, impacts []Impact, most int) ([]Impact, error) {
	for i := range impacts {
		if impacts[i].Outcome != Taken {
			continue
		}
		fortified, err := fortifiedAfter(ctx, tiles, impacts[i], flag, most)
		if err != nil {
			return impacts, err
		}
		impacts[i] = fortified
	}

	return impacts, nil
}
