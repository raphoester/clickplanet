package clicks

import (
	"context"
	"errors"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

// Infecting is the tiles under the lock of one use or tick of a virus: nothing else reads or writes them until it ends.
type Infecting interface {
	Claimable
	// Virus is the virus as the map holds it: a tile a change of owner cured is gone from it.
	Virus(id VirusID) (Virus, bool)
	// Keep holds the virus as it is now. One that ended leaves the map, and the tiles it infects are cured.
	Keep(virus Virus)
}

type InfectableTiles interface {
	// Infect holds the lock through every write of `write`: one change at one version, then the fortifies it earned, then
	// the knock-outs.
	Infect(ctx context.Context, write func(tiles Infecting) error) error
}

// Pick is a number from 0 to n-1, n above 0.
type Pick func(n int) int

func NewContagion(tiles InfectableTiles, ground Neighbourhood, pick Pick, shieldsPerTile func() int) Contagion {
	return Contagion{tiles: tiles, ground: ground, pick: pick, shieldsPerTile: shieldsPerTile}
}

// Contagion plays a virus on the map: its first take, then each of its ticks, each under one lock of the tiles.
type Contagion struct {
	tiles          InfectableTiles
	ground         Neighbourhood
	pick           Pick
	shieldsPerTile func() int
}

// Start takes the tile the virus is aimed at, which must be an enemy tile with no shield, and infects it.
func (c Contagion) Start(ctx context.Context, virus Virus, most int, next time.Time) (Virus, []Impact, error) {
	shields := c.shieldsPerTile()
	var impacts []Impact
	err := c.tiles.Infect(ctx, func(tiles Infecting) error {
		ctx := cpctx.AddAccountToContext(ctx, virus.account.String())
		tile, err := tiles.Tile(virus.tile)
		if err != nil {
			return err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
		}
		virus, err = virus.Aimed(tile)
		if err != nil {
			return err
		}

		impact, err := claim(ctx, tiles, virus.tile, virus.flag, Claimable.Set, shields)
		if err != nil {
			return err
		}
		virus = virus.Infected(impact).Ticked(true, most, next)
		tiles.Keep(virus)
		impacts, err = knockedOutAll(ctx, tiles, virus.flag, []Impact{impact})

		return err
	})

	return virus, impacts, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
}

// Spread plays one tick of the virus: each tile it infected before infects one enemy tile next to it, of any flag,
// picked at random, until it made its most infections. A tile a shield kept is struck, and stays the other flag's.
func (c Contagion) Spread(ctx context.Context, id VirusID, most int, next time.Time) (Virus, []Impact, error) {
	shields := c.shieldsPerTile()
	var (
		virus   Virus
		impacts []Impact
	)
	err := c.tiles.Infect(ctx, func(tiles Infecting) error {
		var ok bool
		if virus, ok = tiles.Virus(id); !ok {
			return ErrNoVirus
		}
		ctx := cpctx.AddAccountToContext(ctx, virus.account.String())

		// A failure stops the tick, and what it wrote before is kept and told all the same.
		var failed error
		for _, from := range virus.infecting() {
			if virus.spent(most) {
				break
			}
			impact, infected, err := c.infect(ctx, tiles, from, virus, shields)
			if err != nil {
				failed = err
				break
			}
			if infected {
				virus = virus.Infected(impact)
				impacts = append(impacts, impact)
			}
		}

		virus = virus.Ticked(len(impacts) > 0, most, next)
		tiles.Keep(virus)
		var err error
		impacts, err = knockedOutAll(ctx, tiles, virus.flag, impacts)

		return errors.Join(failed, err)
	})

	return virus, impacts, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
}

// infect claims an enemy tile next to the tile; false when it has none.
func (c Contagion) infect(ctx context.Context, tiles Claimable, from uint32, virus Virus, shields int) (Impact, bool, error) {
	target, ok, err := c.target(tiles, from, virus.flag)
	if err != nil || !ok {
		return Impact{}, false, err
	}
	impact, err := claim(ctx, tiles, target, virus.flag, Claimable.Set, shields)
	if err != nil {
		return Impact{}, false, err
	}
	return impact, true, nil
}

// target is a tile of any flag but the virus's next to the tile, picked at random; false for none. A blank tile is none.
func (c Contagion) target(tiles Claimable, from uint32, flag string) (uint32, bool, error) {
	var enemies []uint32
	for _, neighbour := range c.ground.Neighbours(from) {
		tile, err := tiles.Tile(neighbour)
		if err != nil {
			return 0, false, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
		}
		if tile.enemyOf(flag) {
			enemies = append(enemies, neighbour)
		}
	}
	if len(enemies) == 0 {
		return 0, false, nil
	}

	return enemies[c.pick(len(enemies))], true, nil
}

// knockedOutAll asks each take whether it felled a capital once every take is written, as an enclosure does.
func knockedOutAll(ctx context.Context, tiles Claimable, flag string, impacts []Impact) ([]Impact, error) {
	for i := range impacts {
		knocked, err := knockedOut(ctx, tiles, flag, impacts[i])
		if err != nil {
			return impacts, err
		}
		impacts[i] = knocked
	}

	return impacts, nil
}
