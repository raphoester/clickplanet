package bonuses

import "github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"

type BombRules struct {
	Radius float64
	Reach  float64
}

func NewBombRules(config BombConfig, spacing float64) BombRules {
	return BombRules{Radius: config.withDefaults().Rings * spacing, Reach: spacing}
}

type Ground interface {
	Position(id uint32) (clicks.Vec3, bool)
	Within(centre clicks.Vec3, radius float64) []uint32
}

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
