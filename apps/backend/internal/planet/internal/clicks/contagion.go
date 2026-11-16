package clicks

import (
	"context"
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
		impacts, err = knockedOutAll(ctx, tiles, virus.flag, []Impact{impact})
		tiles.Keep(virus)

		return err
	})

	return virus, impacts, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
}

// Spread plays one tick of the virus: each tile it infected before infects one tile of the flag it hit next to it,
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

		for _, from := range virus.infecting() {
			if virus.spent(most) {
				break
			}
			target, ok, err := c.target(tiles, from, virus.victim)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
			impact, err := claim(ctx, tiles, target, virus.flag, Claimable.Set, shields)
			if err != nil {
				return err
			}
			virus = virus.Infected(impact)
			impacts = append(impacts, impact)
		}

		virus = virus.Ticked(len(impacts) > 0, most, next)
		var err error
		impacts, err = knockedOutAll(ctx, tiles, virus.flag, impacts)
		tiles.Keep(virus)

		return err
	})

	return virus, impacts, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
}

// target is a tile of the victim next to the tile, picked at random; false for none.
func (c Contagion) target(tiles Claimable, from uint32, victim string) (uint32, bool, error) {
	var victims []uint32
	for _, neighbour := range c.ground.Neighbours(from) {
		tile, err := tiles.Tile(neighbour)
		if err != nil {
			return 0, false, err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
		}
		if tile.owner == victim {
			victims = append(victims, neighbour)
		}
	}
	if len(victims) == 0 {
		return 0, false, nil
	}

	return victims[c.pick(len(victims))], true, nil
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
