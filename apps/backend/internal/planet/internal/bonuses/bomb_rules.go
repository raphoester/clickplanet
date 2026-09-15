package bonuses

import "github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"

// BombRules is what a bomb is: how wide a circle it clears, and how far from a tile an aim may land and still hit it.
type BombRules struct {
	Radius float64
	Reach  float64
}

// NewBombRules sizes a bomb off the map, so the ring a client draws is the width of what it clears.
func NewBombRules(config BombConfig, spacing float64) BombRules {
	return BombRules{Radius: config.withDefaults().Rings * spacing, Reach: spacing}
}

type Ground interface {
	Position(id uint32) (clicks.Vec3, bool)
	Within(centre clicks.Vec3, radius float64) []uint32
}

// Blast is what a bomb aimed at target clears, given the nearest tile and its arc from the aim.
// Too far from any tile is the sea: nothing is cleared, and the splash is drawn where it was aimed.
func (r BombRules) Blast(country string, target clicks.Vec3, nearest uint32, arc float64, ground Ground) clicks.Blast {
	blast := clicks.Blast{CountryID: country, Radius: r.Radius, Point: target.Unit()}
	if arc > r.Reach {
		return blast
	}

	blast.Tile = nearest
	blast.Point, _ = ground.Position(nearest)
	blast.Cleared = ground.Within(blast.Point, r.Radius)

	return blast
}
