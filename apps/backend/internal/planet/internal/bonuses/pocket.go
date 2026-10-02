package bonuses

import "github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"

type Pocket struct {
	inside []uint32
	wall   []uint32
}

func (p Pocket) Inside() []uint32 {
	return p.inside
}

func (p Pocket) Announcement(country string, closingTile uint32) Enclosed {
	return Enclosed{
		CountryID:   country,
		ClosingTile: closingTile,
		Wall:        p.wall,
		Filled:      p.inside,
	}
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
