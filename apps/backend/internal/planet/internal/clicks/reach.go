package clicks

import (
	"errors"
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

var ErrOutOfReach = errors.New("the tile touches none of the flag's land and is on the coast of none of its waters")

type Neighbourhood interface {
	Neighbours(id uint32) []uint32
}

type Shores interface {
	BodiesOf(tile uint32) []BodyID
	CoastOf(body BodyID) []uint32
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

// Reach is where a flag's take lands while the rules in force hold it: on its own land, on a tile next to it, or on the
// coast of a water where it holds a coast. Otherwise, everywhere.
type Reach struct {
	on         func() Reaching
	neighbours Neighbourhood
	waters     Shores
	held       Holdings
}

func NewReach(on func() Reaching, neighbours Neighbourhood, waters Shores, held Holdings) Reach {
	return Reach{on: on, neighbours: neighbours, waters: waters, held: held}
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
	if r.held.HoldsAny(flag, []uint32{tile}) || r.held.HoldsAny(flag, r.neighbours.Neighbours(tile)) {
		return true
	}
	for _, body := range r.waters.BodiesOf(tile) {
		if r.held.HoldsAny(flag, r.waters.CoastOf(body)) {
			return true
		}
	}

	return false
}
