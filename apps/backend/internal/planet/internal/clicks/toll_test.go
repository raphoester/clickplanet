package clicks_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type shares map[string]float64

func (s shares) Share(country string) float64 { return s[country] }

var table = clicks.TollConfig{Steps: []clicks.TollStep{{Share: 0.1, Slowdown: 1.5}, {Share: 0.5, Slowdown: 2}}}

func TestPriceClimbsTheSteps(t *testing.T) {
	pricer := clicks.NewToll(table, shares{"small": 0.05, "edge": 0.1, "mid": 0.3, "top": 0.8})

	assert.Equal(t, clicks.Price{Slowdown: 1, Share: 0.05, NextShare: 0.1, NextSlowdown: 1.5}, pricer.Price("small"))
	assert.Equal(t, clicks.Price{Slowdown: 1.5, Share: 0.1, NextShare: 0.5, NextSlowdown: 2}, pricer.Price("edge"), "a step starts at its share")
	assert.Equal(t, clicks.Price{Slowdown: 1.5, Share: 0.3, NextShare: 0.5, NextSlowdown: 2}, pricer.Price("mid"))
	assert.Equal(t, clicks.Price{Slowdown: 2, Share: 0.8}, pricer.Price("top"), "nothing comes after the top step")
	assert.InDelta(t, 1, pricer.Price("unknown").Slowdown, 1e-9)
}

func TestNoStepsRefillsEveryCountryAtThePlainRate(t *testing.T) {
	assert.Equal(t, clicks.Price{Slowdown: 1, Share: 0.9}, clicks.NewToll(clicks.TollConfig{}, shares{"bg": 0.9}).Price("bg"))
}

func TestValidate(t *testing.T) {
	require.NoError(t, table.Validate())
	require.NoError(t, clicks.TollConfig{}.Validate())

	for name, config := range map[string]clicks.TollConfig{
		"a slowdown of one":               {Steps: []clicks.TollStep{{Share: 0.1, Slowdown: 1}}},
		"a slowdown that is not a number": {Steps: []clicks.TollStep{{Share: 0.1, Slowdown: math.NaN()}}},
		"a falling slowdown":              {Steps: []clicks.TollStep{{Share: 0.1, Slowdown: 1.5}, {Share: 0.2, Slowdown: 1.25}}},
		"a falling share":                 {Steps: []clicks.TollStep{{Share: 0.5, Slowdown: 2}, {Share: 0.2, Slowdown: 3}}},
		"a share of zero":                 {Steps: []clicks.TollStep{{Share: 0, Slowdown: 2}}},
		"a share above the map":           {Steps: []clicks.TollStep{{Share: 1.5, Slowdown: 2}}},
		"an infinite slowdown":            {Steps: []clicks.TollStep{{Share: 0.5, Slowdown: math.Inf(1)}}},
	} {
		t.Run(name, func(t *testing.T) {
			require.Error(t, config.Validate())
		})
	}
}
