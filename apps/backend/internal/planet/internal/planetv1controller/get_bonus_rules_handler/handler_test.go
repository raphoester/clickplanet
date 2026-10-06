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

func TestTheRulesAreAnsweredAsTheyWereGiven(t *testing.T) {
	rules := bonuses.Rules{BlastRadius: 0.016, EnclosureMaxTiles: 25, SpreadClicks: 8, Defenders: 12, TileDefenders: 10}
	res, err := get_bonus_rules_handler.New(rules, toll{}).
		GetBonusRules(t.Context(), connect.NewRequest(&planetv1.GetBonusRulesRequest{}))
	require.NoError(t, err)

	assert.InDelta(t, 0.016, res.Msg.GetBlastRadius(), 1e-9)
	assert.Equal(t, uint32(25), res.Msg.GetEnclosureMaxTiles())
	assert.Equal(t, uint32(8), res.Msg.GetSpreadClicks())
	assert.Equal(t, uint32(12), res.Msg.GetDefenders())
	assert.Equal(t, uint32(10), res.Msg.GetTileDefenders())
	assert.Empty(t, res.Msg.GetTollSteps())
}

func TestTheClientIsToldEveryTollStepInOrder(t *testing.T) {
	steps := toll{{Share: 0.1, Slowdown: 1.5}, {Share: 0.2, Slowdown: 2.5}, {Share: 0.3, Slowdown: 4}}

	res, err := get_bonus_rules_handler.New(bonuses.Rules{}, steps).
		GetBonusRules(t.Context(), connect.NewRequest(&planetv1.GetBonusRulesRequest{}))
	require.NoError(t, err)

	got := res.Msg.GetTollSteps()
	require.Len(t, got, 3)
	for i, step := range steps {
		assert.InDelta(t, step.Share, got[i].GetShare(), 1e-9)
		assert.InDelta(t, step.Slowdown, got[i].GetSlowdown(), 1e-9)
	}
}
