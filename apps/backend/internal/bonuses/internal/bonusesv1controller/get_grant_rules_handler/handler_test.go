package get_grant_rules_handler_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/get_grant_rules_handler"
)

func TestTheAnswerFollowsTheSetInForce(t *testing.T) {
	rules := bonuses.TestRules()
	handler := get_grant_rules_handler.New(func() bonuses.Rules { return rules })

	for _, refill := range []bool{true, false} {
		rules.SignInRefill = refill

		res, err := handler.GetGrantRules(t.Context(), connect.NewRequest(&bonusesv1.GetGrantRulesRequest{}))
		require.NoError(t, err)
		assert.Equal(t, refill, res.Msg.GetSignInRefill())
	}
}

func TestTheAnswerTellsTheGatesAndTheKindsOpenToEveryoneInPlanetsWords(t *testing.T) {
	handler := get_grant_rules_handler.New(bonuses.TestGated)

	res, err := handler.GetGrantRules(t.Context(), connect.NewRequest(&bonusesv1.GetGrantRulesRequest{}))
	require.NoError(t, err)

	assert.True(t, res.Msg.GetGatesOn())
	assert.ElementsMatch(t, []planetv1.BonusKind{
		planetv1.BonusKind_BONUS_KIND_REFILL, planetv1.BonusKind_BONUS_KIND_SPREAD_CLICKS, planetv1.BonusKind_BONUS_KIND_SHIELDS,
	}, res.Msg.GetOpenKinds())
}

func TestTheGatesAreOffUntilTheLevelsAreShown(t *testing.T) {
	handler := get_grant_rules_handler.New(bonuses.TestRules)

	res, err := handler.GetGrantRules(t.Context(), connect.NewRequest(&bonusesv1.GetGrantRulesRequest{}))
	require.NoError(t, err)

	assert.False(t, res.Msg.GetGatesOn())
}
