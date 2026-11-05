package clicks_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type shares map[string]float64

func (s shares) Share(country string) float64 { return s[country] }

func table() clicks.TollSteps {
	return clicks.TollSteps{{Share: 0.1, Slowdown: 1.5}, {Share: 0.5, Slowdown: 2}}
}

func TestPriceClimbsTheSteps(t *testing.T) {
	pricer := clicks.NewToll(table, shares{"small": 0.05, "edge": 0.1, "mid": 0.3, "top": 0.8})

	assert.Equal(t, clicks.Price{Slowdown: 1, Share: 0.05, NextShare: 0.1, NextSlowdown: 1.5}, pricer.Price("small"))
	assert.Equal(t, clicks.Price{Slowdown: 1.5, Share: 0.1, NextShare: 0.5, NextSlowdown: 2}, pricer.Price("edge"), "a step starts at its share")
	assert.Equal(t, clicks.Price{Slowdown: 1.5, Share: 0.3, NextShare: 0.5, NextSlowdown: 2}, pricer.Price("mid"))
	assert.Equal(t, clicks.Price{Slowdown: 2, Share: 0.8}, pricer.Price("top"), "nothing comes after the top step")
	assert.InDelta(t, 1, pricer.Price("unknown").Slowdown, 1e-9)
}

func TestNoStepsRefillsEveryCountryAtThePlainRate(t *testing.T) {
	assert.Equal(t, clicks.Price{Slowdown: 1, Share: 0.9}, clicks.NewToll(func() clicks.TollSteps { return nil }, shares{"bg": 0.9}).Price("bg"))
}

func TestTheTollFollowsTheStepsInForce(t *testing.T) {
	steps := clicks.TollSteps{{Share: 0.5, Slowdown: 2}}
	pricer := clicks.NewToll(func() clicks.TollSteps { return steps }, shares{"bg": 0.3})

	steps = clicks.TollSteps{{Share: 0.2, Slowdown: 3}}

	assert.InDelta(t, 3.0, pricer.Price("bg").Slowdown, 1e-9)
	assert.Equal(t, []clicks.TollStep(steps), pricer.Steps())
}
