package get_charge_rules_handler_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_charge_rules_handler"
)

type toll []clicks.TollStep

func (t toll) Steps() []clicks.TollStep { return t }

func noReach() clicks.Reaching { return false }

func noCapitalRule() clicks.CapitalRules { return clicks.CapitalRules{} }

func answered(t *testing.T, handler get_charge_rules_handler.GetChargeRulesHandler) *planetv1.GetChargeRulesResponse {
	t.Helper()

	res, err := handler.GetChargeRules(t.Context(), connect.NewRequest(&planetv1.GetChargeRulesRequest{}))
	require.NoError(t, err)
	return res.Msg
}

func TestTheRulesAreThoseInForce(t *testing.T) {
	rules := charges.TestRules()
	inForce := func() charges.Rules { return rules }
	handler := get_charge_rules_handler.New(inForce, charges.NewBombing(inForce, 0.004), toll{}, noReach, noCapitalRule)

	rules.Bomb.Rings = 5
	rules.Enclose = charges.EncloseRules{MaxTiles: 40, Held: 4, Unlimited: true}
	rules.Spread = charges.SpreadRules{Clicks: 9, Rings: 2, Unlimited: true}
	rules.Shield.Held = 12
	rules.Shield.PerTile = 20
	rules.Bomb.Off = true
	got := answered(t, handler)

	assert.InDelta(t, 0.02, got.GetBlastRadius(), 1e-9)
	assert.Equal(t, uint32(40), got.GetEnclosureMaxTiles())
	assert.Equal(t, uint32(4), got.GetEnclosures())
	assert.True(t, got.GetEnclosuresUnlimited())
	assert.Equal(t, uint32(9), got.GetSpreadClicks())
	assert.Equal(t, uint32(2), got.GetSpreadRings())
	assert.True(t, got.GetSpreadUnlimited())
	assert.Equal(t, uint32(12), got.GetShields())
	assert.Equal(t, uint32(20), got.GetTileShields())
	assert.True(t, got.GetBombsOff())
	assert.Empty(t, got.GetTollSteps())
}

func TestTheClientIsToldEveryTollStepInOrder(t *testing.T) {
	steps := toll{{Share: 0.1, Slowdown: 1.5}, {Share: 0.2, Slowdown: 2.5}, {Share: 0.3, Slowdown: 4}}
	got := answered(t, get_charge_rules_handler.New(charges.TestRules, charges.NewBombing(charges.TestRules, 0.004), steps,
		noReach, noCapitalRule)).GetTollSteps()

	require.Len(t, got, 3)
	for i, step := range steps {
		assert.InDelta(t, step.Share, got[i].GetShare(), 1e-9)
		assert.InDelta(t, step.Slowdown, got[i].GetSlowdown(), 1e-9)
	}
}

func TestThePageIsToldTheReachAndTheCapitalRulesInForce(t *testing.T) {
	reach := clicks.Reaching(false)
	capitals := clicks.CapitalRules{}
	handler := get_charge_rules_handler.New(charges.TestRules, charges.NewBombing(charges.TestRules, 0.004), toll{},
		func() clicks.Reaching { return reach }, func() clicks.CapitalRules { return capitals })

	off := answered(t, handler)
	reach, capitals = true, clicks.CapitalRules{KnockOut: true, FixedShields: true}
	on := answered(t, handler)

	assert.False(t, off.GetReach() || off.GetKnockOut() || off.GetCapitalShieldsFixed())
	assert.True(t, on.GetReach())
	assert.True(t, on.GetKnockOut())
	assert.True(t, on.GetCapitalShieldsFixed())
}
