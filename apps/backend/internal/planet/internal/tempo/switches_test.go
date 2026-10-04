package tempo_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
)

type flatToll struct{}

func (flatToll) Price(string) clicks.Price { return clicks.Price{Slowdown: 1.5, Share: 0.12} }

func TestSwitchesStartPlainAndHoldTheLastRulesSet(t *testing.T) {
	switches := tempo.NewSwitches()
	assert.Equal(t, tempo.Plain(), switches.Rules())

	rules, err := tempo.NewRules(3, time.Minute, true)
	require.NoError(t, err)
	switches.Set(rules)

	assert.Equal(t, rules, switches.Rules())
}

func TestThePricingCarriesTheSpeedupInForce(t *testing.T) {
	switches := tempo.NewSwitches()
	pricing := tempo.NewPricing(flatToll{}, switches)

	assert.Equal(t, clicks.Price{Slowdown: 1.5, Share: 0.12, Speedup: 1}, pricing.Price("fr"))

	rules, err := tempo.NewRules(3, 0, false)
	require.NoError(t, err)
	switches.Set(rules)

	assert.Equal(t, clicks.Price{Slowdown: 1.5, Share: 0.12, Speedup: 3}, pricing.Price("fr"))
}
