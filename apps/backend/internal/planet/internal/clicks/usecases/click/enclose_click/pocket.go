package enclose_click

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click"
)

// Pocket is a closed shape: the tiles inside it, and the owner's tiles around them.
type Pocket struct {
	inside []uint32
	wall   []uint32
}

func (p Pocket) announcement(closing click.In, left int) bonus.Enclosed {
	return bonus.Enclosed{
		CountryID:   closing.CountryID,
		ClosingTile: closing.TileID,
		Wall:        p.wall,
		Filled:      p.inside,
		Left:        left,
	}
}

// pocketBuilder grows one pocket, breadth first from the tile next to the click.
type pocketBuilder struct {
	pocket   Pocket
	inside   map[uint32]bool
	wall     map[uint32]bool
	maxTiles int
}

func newPocketBuilder(start uint32, maxTiles int) *pocketBuilder {
	return &pocketBuilder{
		pocket:   Pocket{inside: []uint32{start}},
		inside:   map[uint32]bool{start: true},
		wall:     map[uint32]bool{},
		maxTiles: maxTiles,
	}
}

func (b *pocketBuilder) knows(tile uint32) bool {
	return b.inside[tile] || b.wall[tile]
}

func (b *pocketBuilder) addWall(tile uint32) {
	b.wall[tile] = true
	b.pocket.wall = append(b.pocket.wall, tile)
}

// addInside refuses a tile past the limit: the shape is too big, or open.
func (b *pocketBuilder) addInside(tile uint32) bool {
	if len(b.pocket.inside) == b.maxTiles {
		return false
	}

	b.inside[tile] = true
	b.pocket.inside = append(b.pocket.inside, tile)

	return true
}

func (b *pocketBuilder) tile(index int) (uint32, bool) {
	if index >= len(b.pocket.inside) {
		return 0, false
	}

	return b.pocket.inside[index], true
}
