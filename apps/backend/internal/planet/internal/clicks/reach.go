package clicks

import (
	"errors"
	"fmt"

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
	if !bool(r.on()) || r.reaches(flag, tile) {
		return nil
	}

	return fmt.Errorf("%w: tile %d for %s", ErrOutOfReach, tile, flag)
}

// Around is the tile's neighbours a take of the flag lands on. Without the reach it is the map's own table: read it,
// never write it.
func (r Reach) Around(flag string, tile uint32) []uint32 {
	neighbours := r.neighbours.Neighbours(tile)
	if !r.on() {
		return neighbours
	}

	reached := make([]uint32, 0, len(neighbours))
	for _, neighbour := range neighbours {
		if r.reaches(flag, neighbour) {
			reached = append(reached, neighbour)
		}
	}

	return reached
}

func (r Reach) reaches(flag string, tile uint32) bool {
	return r.held.HoldsAny(flag, []uint32{tile}) ||
		r.held.HoldsAny(flag, r.neighbours.Neighbours(tile)) ||
		r.held.HoldsAny(flag, r.bridges.Across(tile))
}
