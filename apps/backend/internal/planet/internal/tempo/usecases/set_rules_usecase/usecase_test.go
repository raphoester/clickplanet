package set_rules_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo/usecases/set_rules_usecase"
)

var finale = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func TestTheRulesSetAreTheRulesInForce(t *testing.T) {
	switches := tempo.NewSwitches()

	err := set_rules_usecase.New(switches).Execute(t.Context(), set_rules_usecase.In{
		RefillMultiplier: 3, BoxInterval: 2 * time.Minute, GiftTag: "finale-0", GiftMadeBefore: finale,
	})

	require.NoError(t, err)
	rules := switches.Rules()
	assert.InDelta(t, 3.0, rules.RefillMultiplier(), 1e-9)
	interval, _ := rules.BoxInterval()
	assert.Equal(t, 2*time.Minute, interval)
	gift, gifting := rules.Gift()
	require.True(t, gifting)
	assert.Equal(t, tempo.GiftTag("finale-0"), gift.Tag())
	assert.Equal(t, finale, gift.MadeBefore())
	assert.False(t, rules.Frozen())
}

func TestSettingTheRulesAgainReplacesThemWhole(t *testing.T) {
	switches := tempo.NewSwitches()
	useCase := set_rules_usecase.New(switches)
	require.NoError(t, useCase.Execute(t.Context(), set_rules_usecase.In{RefillMultiplier: 3, GiftTag: "finale-0"}))

	require.NoError(t, useCase.Execute(t.Context(), set_rules_usecase.In{Frozen: true}))

	rules := switches.Rules()
	assert.True(t, rules.Frozen())
	assert.InDelta(t, 1.0, rules.RefillMultiplier(), 1e-9)
	_, gifting := rules.Gift()
	assert.False(t, gifting, "a gift not named again is no longer given")
}

func TestRulesNobodyCanPlayByChangeNothing(t *testing.T) {
	switches := tempo.NewSwitches()

	err := set_rules_usecase.New(switches).Execute(t.Context(), set_rules_usecase.In{RefillMultiplier: 0.5, Frozen: true})

	require.ErrorIs(t, err, tempo.ErrInvalidRules)
	assert.Equal(t, tempo.Plain(), switches.Rules())
}
