package charges_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/embedded_board"
)

type ground struct{}

func (g *ground) Position(uint32) (clicks.Vec3, bool) { return clicks.Vec3{Y: 1}, true }

func (g *ground) Within(clicks.Vec3, float64) []uint32 { return []uint32{9, 10, 11} }

func bombOf(rings float64, spacing float64) charges.Bombing {
	rules := charges.TestRules()
	rules.Bomb.Rings = rings
	return charges.NewBombing(func() charges.Rules { return rules }, clicks.Spacing(spacing))
}

var bomb = bombOf(10, 0.003)

func TestABombIsSizedOffTheMap(t *testing.T) {
	assert.InDelta(t, 0.03, bomb.Radius(), 1e-9)
	assert.Zero(t, bomb.Blast("fr", clicks.Vec3{X: 1}, 10, 0.0031, &ground{}).Tile, "past one spacing, the aim is in the sea")
}

func TestABombFollowsTheRingsInForce(t *testing.T) {
	rules := charges.TestRules()
	bombing := charges.NewBombing(func() charges.Rules { return rules }, clicks.Spacing(0.003))

	rules.Bomb.Rings = 2

	assert.InDelta(t, 0.006, bombing.Radius(), 1e-9)
}

func TestProductionsBombClearsAboutOneBankInland(t *testing.T) {
	ground, err := embedded_board.Shipped("earth-1")
	require.NoError(t, err)
	geography := ground.Geography()

	rules := bombOf(charges.TestRules().Bomb.Rings, geography.Spacing())

	var cleared []int
	for id := uint32(1); id <= geography.Stats().Tiles; id += 499 {
		if !inland(geography, id) {
			continue
		}
		centre, _ := geography.Position(id)
		cleared = append(cleared, len(geography.Within(centre, rules.Radius())))
	}
	slices.Sort(cleared)

	require.Greater(t, len(cleared), 200)
	assert.InDelta(t, 60, cleared[len(cleared)/2], 5, "the median bomb")
}

func inland(geography *clicks.Geography, id uint32) bool {
	centre, _ := geography.Position(id)
	for _, tile := range geography.Within(centre, 8*geography.Spacing()) {
		if len(geography.Neighbours(tile)) != clicks.MaxDegree {
			return false
		}
	}

	return true
}
