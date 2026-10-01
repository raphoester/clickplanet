package clicks_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type shares map[string]float64

func (s shares) Share(country string) float64 { return s[country] }

// flags is every tally, set by the test.
type flags map[clicks.AllegianceKey]clicks.Allegiance

func (f flags) Allegiance(key clicks.AllegianceKey) clicks.Allegiance { return f[key] }

var epoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

var (
	table = clicks.TollConfig{Steps: []clicks.TollStep{{Share: 0.1, Slowdown: 1.5}, {Share: 0.5, Slowdown: 2}}}
	board = shares{"small": 0.05, "edge": 0.1, "mid": 0.3, "top": 0.8}
)

func tollOver(flags flags) *clicks.Toll {
	return clicks.NewToll(table, board, flags, cptime.NewFixedClock(epoch))
}

func paintedFor(country string, times int) clicks.Allegiance {
	allegiance := clicks.Allegiance{}
	for range times {
		allegiance = allegiance.With(country, epoch)
	}

	return allegiance
}

func TestPriceClimbsTheSteps(t *testing.T) {
	pricer := tollOver(flags{})

	assert.Equal(t, clicks.Price{Country: "small", Slowdown: 1, Share: 0.05, NextShare: 0.1, NextSlowdown: 1.5}, pricer.Price("small"))
	assert.Equal(t, clicks.Price{Country: "edge", Slowdown: 1.5, Share: 0.1, NextShare: 0.5, NextSlowdown: 2}, pricer.Price("edge"), "a step starts at its share")
	assert.Equal(t, clicks.Price{Country: "mid", Slowdown: 1.5, Share: 0.3, NextShare: 0.5, NextSlowdown: 2}, pricer.Price("mid"))
	assert.Equal(t, clicks.Price{Country: "top", Slowdown: 2, Share: 0.8}, pricer.Price("top"), "nothing comes after the top step")
	assert.InDelta(t, 1, pricer.Price("unknown").Slowdown, 1e-9)
}

func TestNoStepsRefillsEveryCountryAtThePlainRate(t *testing.T) {
	toll := clicks.NewToll(clicks.TollConfig{}, shares{"bg": 0.9}, flags{}, cptime.NewFixedClock(epoch))

	assert.Equal(t, clicks.Price{Country: "bg", Slowdown: 1, Share: 0.9}, toll.Price("bg"))
}

func TestSwitchingToASmallFlagKeepsTheMainFlagsPrice(t *testing.T) {
	pricer := tollOver(flags{clicks.AccountAllegianceKey("acc"): paintedFor("top", 60)})

	price := pricer.PriceFor(clicks.Payer{Scope: "2001:db8::/64", Account: "acc"}, "small")

	assert.Equal(t, "top", price.Country)
	assert.InDelta(t, 2, price.Slowdown, 1e-9, "a bank spent on a big flag refills at the big flag's pace")
}

func TestAPayerWithNoHistoryIsPricedByTheCountryItClicksFor(t *testing.T) {
	price := tollOver(flags{}).PriceFor(clicks.Payer{Scope: "2001:db8::/64", Account: "new"}, "mid")

	assert.Equal(t, clicks.Price{Country: "mid", Slowdown: 1.5, Share: 0.3, NextShare: 0.5, NextSlowdown: 2}, price)
}

func TestAPayerWithNoAccountIsPricedByItsScopesFlag(t *testing.T) {
	pricer := tollOver(flags{clicks.ScopeAllegianceKey("2001:db8::/64"): paintedFor("top", 60)})

	assert.Equal(t, "top", pricer.PriceFor(clicks.Payer{Scope: "2001:db8::/64"}, "small").Country)
	assert.Equal(t, "small", pricer.PriceFor(clicks.Payer{Scope: "2001:db8::/64", Account: "acc"}, "small").Country,
		"an account is priced by its own flag, not by the address it plays from")
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
