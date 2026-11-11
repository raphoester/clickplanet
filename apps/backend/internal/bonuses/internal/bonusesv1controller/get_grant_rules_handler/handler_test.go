package get_grant_rules_handler_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
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
