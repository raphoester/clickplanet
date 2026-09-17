package bonuses

import "github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"

// Pocket is a closed shape: the tiles inside it, and the owner's tiles around them.
type Pocket struct {
	inside []uint32
	wall   []uint32
}

func (p Pocket) Inside() []uint32 {
	return p.inside
}

// Announcement is the shape as the planet is told of it, closed by a click for country on closingTile.
func (p Pocket) Announcement(country string, closingTile uint32, left int) Enclosed {
	return Enclosed{
		CountryID:   country,
		ClosingTile: closingTile,
		Wall:        p.wall,
		Filled:      p.inside,
		Left:        left,
	}
}

// pocketBuilder grows one pocket, breadth first from the tile next to the click.
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

// addInside refuses a tile past the limit: the shape is too big, or open.
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
