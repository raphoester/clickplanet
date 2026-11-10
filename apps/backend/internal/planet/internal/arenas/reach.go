package arenas

import (
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var ErrOutOfReach = errors.New("the tile touches none of the flag's land and is on the coast of none of its waters")

type Neighbourhood interface {
	Neighbours(id uint32) []uint32
}

type Shores interface {
	BodiesOf(tile uint32) []clicks.BodyID
	CoastOf(body clicks.BodyID) []uint32
}

type Holdings interface {
	HoldsAny(country string, tiles []uint32) bool
}

// Reach is where a flag's take lands while an arena plays: on its own land, on a tile next to it, or on the coast of a
// water where it holds a coast. Outside an arena, everywhere.
type Reach struct {
	arena      *InPlay
	neighbours Neighbourhood
	waters     Shores
	held       Holdings
}

func NewReach(arena *InPlay, neighbours Neighbourhood, waters Shores, held Holdings) Reach {
	return Reach{arena: arena, neighbours: neighbours, waters: waters, held: held}
}

func (r Reach) Refuses(flag string, tile uint32) error {
	if !r.arena.Playing() || r.reaches(flag, tile) {
		return nil
	}

	return fmt.Errorf("%w: tile %d for %s", ErrOutOfReach, tile, flag)
}

// Around is the tile's neighbours a take of the flag lands on. Outside an arena it is the map's own table: read it,
// never write it.
func (r Reach) Around(flag string, tile uint32) []uint32 {
	neighbours := r.neighbours.Neighbours(tile)
	if !r.arena.Playing() {
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
