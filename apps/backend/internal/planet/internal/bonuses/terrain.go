package bonuses

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type Neighbours interface {
	Neighbours(id uint32) []uint32
}

type Owners interface {
	Owner(tile uint32) (string, bool)
}

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

func (t Terrain) PocketsClosedBy(tile uint32, country string, maxTiles int) []Pocket {
	search := pocketSearch{terrain: t, country: country, maxTiles: maxTiles, seen: cpcolls.NewSet[uint32]()}
	return search.around(tile)
}

func (t Terrain) onEdge(tile uint32) bool {
	return len(t.neighbours.Neighbours(tile)) < clicks.MaxDegree
}

type pocketSearch struct {
	terrain  Terrain
	country  string
	maxTiles int

	seen *cpcolls.Set[uint32]
}

func (s *pocketSearch) around(tile uint32) []Pocket {
	var pockets []Pocket

	for _, start := range s.terrain.neighbours.Neighbours(tile) {
		if s.seen.Contains(start) || s.terrain.Holds(start, s.country) {
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
	s.seen.Add(start)

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
				s.seen.Add(next)
			}
		}
	}
}
