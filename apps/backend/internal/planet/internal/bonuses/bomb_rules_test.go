package bonuses_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type ground struct {
	centre clicks.Vec3
	radius float64
}

func (g *ground) Position(uint32) (clicks.Vec3, bool) { return clicks.Vec3{Y: 1}, true }

func (g *ground) Within(centre clicks.Vec3, radius float64) []uint32 {
	g.centre, g.radius = centre, radius
	return []uint32{9, 10, 11}
}

var bomb = bonuses.NewBombRules(10, 0.003)

func TestABombIsSizedOffTheMap(t *testing.T) {
	assert.InDelta(t, 0.03, bomb.Radius, 1e-9)
	assert.InDelta(t, 0.003, bomb.Reach, 1e-9)
}

func TestABombOnLandClearsACircleAroundTheTileHit(t *testing.T) {
	area := &ground{}

	blast := bomb.Blast("fr", clicks.Vec3{X: 2}, 10, 0.001, area)

	assert.Equal(t, clicks.Blast{
		Tile: 10, CountryID: "fr", Point: clicks.Vec3{Y: 1}, Radius: bomb.Radius, Cleared: []uint32{9, 10, 11},
	}, blast, "drawn at the tile's centre")
	assert.Equal(t, clicks.Vec3{Y: 1}, area.centre, "the circle is centred on the tile hit, not the raw aim")
	assert.InDelta(t, bomb.Radius, area.radius, 1e-9)
}

func TestABombInTheSeaClearsNothing(t *testing.T) {
	blast := bomb.Blast("fr", clicks.Vec3{X: 2}, 10, 0.2, &ground{})

	assert.Equal(t, clicks.Blast{CountryID: "fr", Point: clicks.Vec3{X: 1}, Radius: bomb.Radius}, blast,
		"drawn where it was aimed, on the unit sphere")
}
