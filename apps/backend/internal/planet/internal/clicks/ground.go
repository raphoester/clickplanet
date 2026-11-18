package clicks

import (
	"errors"
	"fmt"
	"sync/atomic"
)

var ErrNoGround = errors.New("no board plays yet")

// Ground is one board's geometry as the rules read it: which tiles touch and where each is, whose ground each is, the
// bridges, the capitals and the provinces. A tile id means a place only on its board, so a rule reads the ground of the board its map
// plays (Played).
type Ground struct {
	name      BoardName
	geography *Geography
	borders   *Borders
	bridges   *Bridges
	capitals  *Capitals
	provinces *Provinces
	spacing   float64
}

// NewGround refuses files of another size than the geography's: they are another board's. The spacing is measured once.
func NewGround(
	name BoardName, geography *Geography, borders *Borders, bridges *Bridges, capitals *Capitals, provinces *Provinces,
) (*Ground, error) {
	tiles := geography.Stats().Tiles
	if borders.Tiles() != tiles {
		return nil, fmt.Errorf("board %s: the borders hold %d tiles, the geography %d", name, borders.Tiles(), tiles)
	}
	if bridges.Tiles() != tiles {
		return nil, fmt.Errorf("board %s: the bridges hold %d tiles, the geography %d", name, bridges.Tiles(), tiles)
	}
	return &Ground{
		name: name, geography: geography, borders: borders, bridges: bridges, capitals: capitals, provinces: provinces,
		spacing: geography.Spacing(),
	}, nil
}

func (g *Ground) Name() BoardName       { return g.name }
func (g *Ground) Geography() *Geography { return g.geography }
func (g *Ground) Borders() *Borders     { return g.borders }
func (g *Ground) Bridges() *Bridges     { return g.bridges }
func (g *Ground) Capitals() *Capitals   { return g.capitals }
func (g *Ground) Provinces() *Provinces { return g.provinces }
func (g *Ground) Tiles() uint32         { return g.geography.Stats().Tiles }
func (g *Ground) Spacing() float64      { return g.spacing }

// Grounds are the boards this process carries, by planet's row of each, each loaded when first needed.
type Grounds interface {
	GroundOf(board BoardID) (*Ground, error)
}

// Played is the ground of the board the map in play plays. The tile map swaps it with its stage, under the lock a write
// and its record hold, so every rule reads the board its map plays: the world's, or an arena's of its own (arena.board).
type Played struct {
	ground atomic.Pointer[Ground]
}

func NewPlayed() *Played {
	return &Played{}
}

// Play makes ground the one in play.
func (p *Played) Play(ground *Ground) {
	p.ground.Store(ground)
}

// Ground is the ground in play: nil before the tile map loads.
func (p *Played) Ground() *Ground {
	return p.ground.Load()
}

func (p *Played) in() *Ground {
	ground := p.ground.Load()
	if ground == nil {
		panic(ErrNoGround)
	}
	return ground
}

func (p *Played) Name() BoardName { return p.in().name }

// Tiles is the board's tile count: its last tile id.
func (p *Played) Tiles() uint32 { return p.in().Tiles() }

func (p *Played) Spacing() float64 { return p.in().spacing }

func (p *Played) Neighbours(id uint32) []uint32 { return p.in().geography.Neighbours(id) }

func (p *Played) Position(id uint32) (Vec3, bool) { return p.in().geography.Position(id) }

func (p *Played) Within(centre Vec3, radius float64) []uint32 {
	return p.in().geography.Within(centre, radius)
}

func (p *Played) Nearest(point Vec3) (uint32, float64) { return p.in().geography.Nearest(point) }

// CountryOf is the country whose ground the tile is.
func (p *Played) CountryOf(tile uint32) string { return p.in().borders.CountryOf(tile) }

func (p *Played) CountryOfLandmass(landmass LandmassID) string {
	return p.in().borders.CountryOfLandmass(landmass)
}

func (p *Played) LandmassOf(tile uint32) LandmassID { return p.in().borders.LandmassOf(tile) }

func (p *Played) TilesOf(landmass LandmassID) []uint32 { return p.in().borders.TilesOf(landmass) }

func (p *Played) CountrySizes() []CountrySize { return p.in().borders.CountrySizes() }

func (p *Played) Landmasses() int { return p.in().borders.Landmasses() }

func (p *Played) Across(tile uint32) []uint32 { return p.in().bridges.Across(tile) }

func (p *Played) Capitals() *Capitals { return p.in().capitals }

// CheckTile is true for a tile of the board in play.
func (p *Played) CheckTile(tile uint32) bool {
	return tile > 0 && tile <= p.Tiles()
}

// MaxIndex is the last tile of the board in play.
func (p *Played) MaxIndex() uint32 {
	return p.Tiles()
}

// OffMap is true for a page load that does not read from tile 1, or clamps its last batch past the board.
func (p *Played) OffMap(start, end uint32) bool {
	return start == 0 || end > p.Tiles()
}
