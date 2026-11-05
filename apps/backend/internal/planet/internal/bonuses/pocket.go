package bonuses

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type Pocket struct {
	inside []uint32
	wall   []uint32
}

func (p Pocket) Inside() []uint32 {
	return p.inside
}

func (p Pocket) Empty() bool {
	return len(p.inside) == 0
}

func (p Pocket) Enclosure(country string, closingTile uint32) clicks.Enclosure {
	return clicks.Enclosure{Country: country, ClosingTile: closingTile, Wall: p.wall}
}

func joined(shapes []Pocket) Pocket {
	if len(shapes) == 1 {
		return shapes[0]
	}

	var pocket Pocket
	walls := cpcolls.NewSet[uint32]()
	for _, shape := range shapes {
		for _, tile := range shape.wall {
			if !walls.Contains(tile) {
				walls.Add(tile)
				pocket.wall = append(pocket.wall, tile)
			}
		}
	}

	pocket.inside = make([]uint32, 0, insideOf(shapes))
	for index := 0; len(pocket.inside) < cap(pocket.inside); index++ {
		for _, shape := range shapes {
			if index < len(shape.inside) {
				pocket.inside = append(pocket.inside, shape.inside[index])
			}
		}
	}

	return pocket
}

func insideOf(shapes []Pocket) int {
	total := 0
	for _, shape := range shapes {
		total += len(shape.inside)
	}

	return total
}

type pocketBuilder struct {
	pocket   Pocket
	inside   *cpcolls.Set[uint32]
	wall     *cpcolls.Set[uint32]
	maxTiles int
}

func newPocketBuilder(start uint32, maxTiles int) *pocketBuilder {
	return &pocketBuilder{
		pocket:   Pocket{inside: []uint32{start}},
		inside:   cpcolls.NewSet(start),
		wall:     cpcolls.NewSet[uint32](),
		maxTiles: maxTiles,
	}
}

func (b *pocketBuilder) knows(tile uint32) bool {
	return b.inside.Contains(tile) || b.wall.Contains(tile)
}

func (b *pocketBuilder) addWall(tile uint32) {
	b.wall.Add(tile)
	b.pocket.wall = append(b.pocket.wall, tile)
}

func (b *pocketBuilder) addInside(tile uint32) bool {
	if len(b.pocket.inside) == b.maxTiles {
		return false
	}

	b.inside.Add(tile)
	b.pocket.inside = append(b.pocket.inside, tile)

	return true
}

func (b *pocketBuilder) tile(index int) (uint32, bool) {
	if index >= len(b.pocket.inside) {
		return 0, false
	}

	return b.pocket.inside[index], true
}
