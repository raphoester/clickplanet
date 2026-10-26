package get_bonus_rules_handler_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_bonus_rules_handler"
)

type toll []clicks.TollStep

func (t toll) Steps() []clicks.TollStep { return t }

func answered(t *testing.T, handler get_bonus_rules_handler.GetBonusRulesHandler) *planetv1.GetBonusRulesResponse {
	t.Helper()

	res, err := handler.GetBonusRules(t.Context(), connect.NewRequest(&planetv1.GetBonusRulesRequest{}))
	require.NoError(t, err)
	return res.Msg
}

func TestTheRulesAreThoseInForce(t *testing.T) {
	rules := &bonuses.FixedRules{Rules: bonuses.TestRules()}
	handler := get_bonus_rules_handler.New(rules, bonuses.NewBombing(rules, 0.004), toll{})

	rules.Rules.Bomb.Rings = 5
	rules.Rules.Enclose = bonuses.EncloseRules{MaxTiles: 40, Held: 4, MaxPerBox: 2}
	rules.Rules.Spread.Clicks = 9
	rules.Rules.Shield.Held = 12
	rules.Rules.Shield.PerTile = 20
	got := answered(t, handler)

	assert.InDelta(t, 0.02, got.GetBlastRadius(), 1e-9)
	assert.Equal(t, uint32(40), got.GetEnclosureMaxTiles())
	assert.Equal(t, uint32(4), got.GetEnclosures())
	assert.Equal(t, uint32(9), got.GetSpreadClicks())
	assert.Equal(t, uint32(12), got.GetShields())
	assert.Equal(t, uint32(20), got.GetTileShields())
	assert.Empty(t, got.GetTollSteps())
}

func TestTheClientIsToldEveryTollStepInOrder(t *testing.T) {
	steps := toll{{Share: 0.1, Slowdown: 1.5}, {Share: 0.2, Slowdown: 2.5}, {Share: 0.3, Slowdown: 4}}
	rules := &bonuses.FixedRules{Rules: bonuses.TestRules()}

	got := answered(t, get_bonus_rules_handler.New(rules, bonuses.NewBombing(rules, 0.004), steps)).GetTollSteps()

	require.Len(t, got, 3)
	for i, step := range steps {
		assert.InDelta(t, step.Share, got[i].GetShare(), 1e-9)
		assert.InDelta(t, step.Slowdown, got[i].GetSlowdown(), 1e-9)
	}
}
