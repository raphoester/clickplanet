package clicks_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// The same cases as the frontend's domain/homeSoil.test.ts: the client paints its own click from its copy of
// the rule, so the two must answer alike.
var homeSoilCases = []struct {
	name                string
	owner, ground, flag string
	outcome             clicks.Outcome
	ownerAfter          string
}{
	{"a native tile clicked for another flag is cleared", "pl", "pl", "de", clicks.Cleared, ""},
	{"a native tile clicked by its natives is unchanged", "pl", "pl", "pl", clicks.Unchanged, "pl"},
	{"an empty tile on home ground is taken by anybody", "", "pl", "de", clicks.Taken, "de"},
	{"an empty tile on home ground is taken back by its natives", "", "pl", "pl", clicks.Taken, "pl"},
	{"a foreign-held tile on home ground is won back by its natives in one", "de", "pl", "pl", clicks.Taken, "pl"},
	{"a foreign-held tile on home ground is taken by a third flag in one", "de", "pl", "fr", clicks.Taken, "fr"},
	{"a tile already wearing the flag is unchanged", "de", "pl", "de", clicks.Unchanged, "de"},
	{"a country's flag on another's ground is taken as always", "pl", "de", "fr", clicks.Taken, "fr"},
	{"a tile in no country is never native", "pl", "", "de", clicks.Taken, "de"},
	{"an empty tile in no country is taken", "", "", "de", clicks.Taken, "de"},
}

func TestOutcomeOf(t *testing.T) {
	for _, c := range homeSoilCases {
		t.Run(c.name, func(t *testing.T) {
			outcome := clicks.OutcomeOf(c.owner, c.ground, c.flag)
			assert.Equal(t, c.outcome, outcome)
			assert.Equal(t, c.ownerAfter, outcome.OwnerAfter(c.owner, c.flag))
		})
	}
}

type grounds map[uint32]string

func (g grounds) CountryOf(tile uint32) string { return g[tile] }

func TestHomeSoilReadsTheTilesOwnGround(t *testing.T) {
	homeSoil := clicks.NewHomeSoil(clicks.HomeSoilConfig{Enabled: true}, grounds{7: "pl", 8: "de"})

	assert.True(t, homeSoil.Enabled())
	assert.Equal(t, clicks.Cleared, homeSoil.Outcome(7, "pl", "de"))
	assert.Equal(t, clicks.Taken, homeSoil.Outcome(8, "pl", "de"), "Poland's flag on German ground is taken")
	assert.Equal(t, clicks.Taken, homeSoil.Outcome(9, "pl", "de"), "a tile in no country is never native")
}

func TestHomeSoilOffTakesEverywhere(t *testing.T) {
	homeSoil := clicks.NewHomeSoil(clicks.HomeSoilConfig{}, grounds{7: "pl"})

	assert.False(t, homeSoil.Enabled())
	assert.Equal(t, clicks.Taken, homeSoil.Outcome(7, "pl", "de"))
	assert.Equal(t, clicks.Unchanged, homeSoil.Outcome(7, "pl", "pl"))
}
