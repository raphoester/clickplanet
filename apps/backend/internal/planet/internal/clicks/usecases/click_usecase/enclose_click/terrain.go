package enclose_click

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Neighbours is the part of clicks.Geography this reads.
type Neighbours interface {
	Neighbours(id uint32) []uint32
}

type Owners interface {
	Owner(tile uint32) (string, bool)
}

// Terrain is the map as the pocket search sees it: who holds a tile, and what touches it.
type Terrain struct {
	neighbours Neighbours
	owners     Owners
}

func NewTerrain(neighbours Neighbours, owners Owners) Terrain {
	return Terrain{neighbours: neighbours, owners: owners}
}

func (t Terrain) Holds(tile uint32, country string) bool {
	owner, _ := t.owners.Owner(tile)
	return owner == country
}

// PocketsClosedBy looks for a small inside rather than an outline: on a sphere,
// every loop has two insides.
func (t Terrain) PocketsClosedBy(tile uint32, country string, maxTiles int) []Pocket {
	search := pocketSearch{terrain: t, country: country, maxTiles: maxTiles, seen: map[uint32]bool{}}
	return search.around(tile)
}

// onEdge is a tile with the sea or a lake beside it. The twelve lattice corners read as one too.
func (t Terrain) onEdge(tile uint32) bool {
	return len(t.neighbours.Neighbours(tile)) < clicks.MaxDegree
}

// pocketSearch floods from each neighbour of a click over tiles the country does not hold.
type pocketSearch struct {
	terrain  Terrain
	country  string
	maxTiles int

	// A flood that passed the limit saw only part of its region, but any other
	// start in that part would pass it too.
	seen map[uint32]bool
}

func (s *pocketSearch) around(tile uint32) []Pocket {
	var pockets []Pocket

	for _, start := range s.terrain.neighbours.Neighbours(tile) {
		if s.seen[start] || s.terrain.Holds(start, s.country) {
			continue
		}

		if pocket, closed := s.flood(start); closed {
			pockets = append(pockets, pocket)
		}
	}

	return pockets
}

func (s *pocketSearch) flood(start uint32) (Pocket, bool) {
	builder := newPocketBuilder(start, s.maxTiles)
	s.seen[start] = true

	for index := 0; ; index++ {
		tile, more := builder.tile(index)
		if !more {
			return builder.pocket, true
		}

		if s.terrain.onEdge(tile) {
			return Pocket{}, false
		}

		for _, next := range s.terrain.neighbours.Neighbours(tile) {
			switch {
			case builder.knows(next):
			case s.terrain.Holds(next, s.country):
				builder.addWall(next)
			default:
				if !builder.addInside(next) {
					return Pocket{}, false
				}
				s.seen[next] = true
			}
		}
	}
}
