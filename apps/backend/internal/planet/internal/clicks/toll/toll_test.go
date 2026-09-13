package toll_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/toll"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

type shares map[string]float64

func (s shares) Share(country string) float64 { return s[country] }

var table = toll.Config{Steps: []toll.Step{{Share: 0.1, Cost: 2}, {Share: 0.5, Cost: 5}}}

func TestPriceClimbsTheSteps(t *testing.T) {
	pricer := toll.New(table, shares{"small": 0.05, "edge": 0.1, "mid": 0.3, "top": 0.8})

	assert.Equal(t, toll.Price{Cost: 1, Share: 0.05, NextShare: 0.1, NextCost: 2}, pricer.Price("small"))
	assert.Equal(t, toll.Price{Cost: 2, Share: 0.1, NextShare: 0.5, NextCost: 5}, pricer.Price("edge"), "a step starts at its share")
	assert.Equal(t, toll.Price{Cost: 2, Share: 0.3, NextShare: 0.5, NextCost: 5}, pricer.Price("mid"))
	assert.Equal(t, toll.Price{Cost: 5, Share: 0.8}, pricer.Price("top"), "nothing comes after the top step")
	assert.Equal(t, 1, pricer.Price("unknown").Cost)
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

func TestValidate(t *testing.T) {
	require.NoError(t, table.Validate(10))
	require.NoError(t, toll.Config{}.Validate(10))

	for name, config := range map[string]toll.Config{
		"a cost of one":          {Steps: []toll.Step{{Share: 0.1, Cost: 1}}},
		"a falling cost":         {Steps: []toll.Step{{Share: 0.1, Cost: 3}, {Share: 0.2, Cost: 2}}},
		"a falling share":        {Steps: []toll.Step{{Share: 0.5, Cost: 2}, {Share: 0.2, Cost: 3}}},
		"a share of zero":        {Steps: []toll.Step{{Share: 0, Cost: 2}}},
		"a share above the map":  {Steps: []toll.Step{{Share: 1.5, Cost: 2}}},
		"a cost above the burst": {Steps: []toll.Step{{Share: 0.5, Cost: 11}}},
	} {
		t.Run(name, func(t *testing.T) {
			require.Error(t, config.Validate(10))
		})
	}
}
