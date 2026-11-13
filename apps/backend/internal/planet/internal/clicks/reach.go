package clicks

import (
	"errors"
	"fmt"
	"slices"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

var ErrOutOfReach = errors.New("the tile touches none of the flag's land and no bridge leads to it from its land")

type Neighbourhood interface {
	Neighbours(id uint32) []uint32
}

type Crossings interface {
	Across(tile uint32) []uint32
}

type Holdings interface {
	HoldsAny(country string, tiles []uint32) bool
}

// Reaching is true while the rules in force hold a take within its flag's reach.
type Reaching bool

func (Reaching) Validate() error { return nil }

var ReachPart = cprules.PartOf[*planetv1.Rules, Reaching]{
	Key: "reach", Read: func(m *planetv1.Rules) Reaching { return Reaching(m.GetReach()) },
}

// Reach is where a flag's take lands while the rules in force hold it: on its own land, on a tile next to it, or at the
// end of a bridge whose other end it holds. Otherwise, everywhere.
type Reach struct {
	on         func() Reaching
	neighbours Neighbourhood
	bridges    Crossings
	held       Holdings
}

func NewReach(on func() Reaching, neighbours Neighbourhood, bridges Crossings, held Holdings) Reach {
	return Reach{on: on, neighbours: neighbours, bridges: bridges, held: held}
}

func (r Reach) Refuses(flag string, tile uint32) error {
	if !bool(r.on()) || r.reachesAny(flag, []uint32{tile}) {
		return nil
	}

	return fmt.Errorf("%w: tile %d for %s", ErrOutOfReach, tile, flag)
}

// Around is the tiles a spread of the flag takes, read after the tile clicked is written: its whole disc once the tile
// is the flag's, and only the tiles in reach when a shield kept it.
func (r Reach) Around(flag string, tile uint32, rings int) []uint32 {
	disc := Disc(r.neighbours, tile, rings)
	if !bool(r.on()) || r.held.HoldsAny(flag, []uint32{tile}) {
		return disc
	}

	return slices.DeleteFunc(disc, func(each uint32) bool { return !r.reachesAny(flag, []uint32{each}) })
}

func (r Reach) reachesAny(flag string, tiles []uint32) bool {
	near := slices.Clone(tiles)
	for _, tile := range tiles {
		near = append(near, r.neighbours.Neighbours(tile)...)
		near = append(near, r.bridges.Across(tile)...)
	}

	return r.held.HoldsAny(flag, near)
}
