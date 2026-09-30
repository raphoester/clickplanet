package get_bonus_rules_handler_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_bonus_rules_handler"
)

// homeSoil is the switch as the file sets it.
type homeSoil bool

func (h homeSoil) Enabled() bool { return bool(h) }

func TestTheRulesAreAnsweredAsTheyWereGiven(t *testing.T) {
	res, err := get_bonus_rules_handler.New(bonuses.Rules{BlastRadius: 0.016, EnclosureMaxTiles: 25, SpreadClicks: 8}, homeSoil(false)).
		GetBonusRules(t.Context(), connect.NewRequest(&planetv1.GetBonusRulesRequest{}))
	require.NoError(t, err)

	assert.InDelta(t, 0.016, res.Msg.GetBlastRadius(), 1e-9)
	assert.Equal(t, uint32(25), res.Msg.GetEnclosureMaxTiles())
	assert.Equal(t, uint32(8), res.Msg.GetSpreadClicks())
	assert.False(t, res.Msg.GetHomeSoil())
}

func TestTheClientIsToldNativeLandTakesTwoClicks(t *testing.T) {
	res, err := get_bonus_rules_handler.New(bonuses.Rules{}, homeSoil(true)).
		GetBonusRules(t.Context(), connect.NewRequest(&planetv1.GetBonusRulesRequest{}))
	require.NoError(t, err)

	assert.True(t, res.Msg.GetHomeSoil())
}
