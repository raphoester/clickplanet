package bonuses

import "github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"

func NewBombing(rules InForce, spacing float64) Bombing {
	return Bombing{rules: rules, spacing: spacing}
}

type Bombing struct {
	rules   InForce
	spacing float64
}

func (b Bombing) Radius() float64 {
	return b.rules.Bonuses().Bomb.Rings * b.spacing
}

type Ground interface {
	Position(id uint32) (clicks.Vec3, bool)
	Within(centre clicks.Vec3, radius float64) []uint32
}

func (b Bombing) Blast(country string, target clicks.Vec3, nearest uint32, arc float64, ground Ground) clicks.Blast {
	blast := clicks.Blast{CountryID: country, Radius: b.Radius(), Point: target.Unit()}
	if arc > b.spacing {
		return blast
	}

	blast.Tile = nearest
	blast.Point, _ = ground.Position(nearest)
	blast.Cleared = ground.Within(blast.Point, blast.Radius)

	return blast
}
