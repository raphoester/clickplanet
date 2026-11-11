package get_grant_rules_handler

import (
	"context"

	"connectrpc.com/connect"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
)

func New(rules func() bonuses.Rules) GetGrantRulesHandler {
	return GetGrantRulesHandler{rules: rules}
}

// GetGrantRulesHandler answers what the page shows of the set the offers play. A GET no cache keeps: a set taken
// reaches an open tab through the stream's rules_changed.
type GetGrantRulesHandler struct {
	rules func() bonuses.Rules
}

func (h GetGrantRulesHandler) GetGrantRules(
	context.Context, *connect.Request[bonusesv1.GetGrantRulesRequest],
) (*connect.Response[bonusesv1.GetGrantRulesResponse], error) {
	return connect.NewResponse(&bonusesv1.GetGrantRulesResponse{SignInRefill: h.rules().SignInRefill}), nil
}
