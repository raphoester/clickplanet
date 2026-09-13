package toll_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/toll"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

type shares map[string]float64

func (s shares) Share(country string) float64 { return s[country] }

var table = toll.Config{Steps: []toll.Step{{Share: 0.1, Cost: 1.5}, {Share: 0.5, Cost: 2}}}

func TestPriceClimbsTheSteps(t *testing.T) {
	pricer := toll.New(table, shares{"small": 0.05, "edge": 0.1, "mid": 0.3, "top": 0.8})

	assert.Equal(t, toll.Price{Cost: 1, Share: 0.05, NextShare: 0.1, NextCost: 1.5}, pricer.Price("small"))
	assert.Equal(t, toll.Price{Cost: 1.5, Share: 0.1, NextShare: 0.5, NextCost: 2}, pricer.Price("edge"), "a step starts at its share")
	assert.Equal(t, toll.Price{Cost: 1.5, Share: 0.3, NextShare: 0.5, NextCost: 2}, pricer.Price("mid"))
	assert.Equal(t, toll.Price{Cost: 2, Share: 0.8}, pricer.Price("top"), "nothing comes after the top step")
	assert.InDelta(t, 1, pricer.Price("unknown").Cost, 1e-9)
}

func TestNoStepsPricesEveryClickAtOne(t *testing.T) {
	assert.Equal(t, toll.Price{Cost: 1, Share: 0.9}, toll.New(toll.Config{}, shares{"bg": 0.9}).Price("bg"))
}

func TestOfCountsTheBucketInClicks(t *testing.T) {
	budget := toll.Of(cpratelimit.State{Tokens: 7, Capacity: 30, PerSecond: 3}, toll.Price{Cost: 3})

	assert.InDelta(t, 7.0/3, budget.Tokens, 1e-9)
	assert.Equal(t, 10, budget.Capacity, "a triple bonus at a cost of three is still ten clicks")
	assert.InDelta(t, 1.0, budget.PerSecond, 1e-9)
}

func TestOfRoundsAFractionalCapacityDown(t *testing.T) {
	budget := toll.Of(cpratelimit.State{Tokens: 10, Capacity: 10, PerSecond: 1}, toll.Price{Cost: 1.5})

	assert.InDelta(t, 10/1.5, budget.Tokens, 1e-9)
	assert.Equal(t, 6, budget.Capacity, "6.67 clicks of room is six whole ones")
	assert.InDelta(t, 1/1.5, budget.PerSecond, 1e-9)
}

func TestValidate(t *testing.T) {
	require.NoError(t, table.Validate(10))
	require.NoError(t, toll.Config{}.Validate(10))

	for name, config := range map[string]toll.Config{
		"a cost of one":               {Steps: []toll.Step{{Share: 0.1, Cost: 1}}},
		"a cost that is not a number": {Steps: []toll.Step{{Share: 0.1, Cost: math.NaN()}}},
		"a falling cost":              {Steps: []toll.Step{{Share: 0.1, Cost: 1.5}, {Share: 0.2, Cost: 1.25}}},
		"a falling share":             {Steps: []toll.Step{{Share: 0.5, Cost: 2}, {Share: 0.2, Cost: 3}}},
		"a share of zero":             {Steps: []toll.Step{{Share: 0, Cost: 2}}},
		"a share above the map":       {Steps: []toll.Step{{Share: 1.5, Cost: 2}}},
		"a cost above the burst":      {Steps: []toll.Step{{Share: 0.5, Cost: 10.5}}},
	} {
		t.Run(name, func(t *testing.T) {
			require.Error(t, config.Validate(10))
		})
	}
}
