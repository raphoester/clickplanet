package get_grant_rules_handler

import (
	"context"

	"connectrpc.com/connect"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/planetwords"
)

func New(rules func() bonuses.Rules) GetGrantRulesHandler {
	return GetGrantRulesHandler{rules: rules}
}

// GetGrantRulesHandler answers what the page shows of the set the bonuses module plays. A GET no cache keeps: the page
// reads it as it loads, and a restart reloads it.
type GetGrantRulesHandler struct {
	rules func() bonuses.Rules
}

func (h GetGrantRulesHandler) GetGrantRules(
	context.Context, *connect.Request[bonusesv1.GetGrantRulesRequest],
) (*connect.Response[bonusesv1.GetGrantRulesResponse], error) {
	rules := h.rules()
	return connect.NewResponse(&bonusesv1.GetGrantRulesResponse{
		SignInRefill: rules.SignInRefill,
		GatesOn:      rules.Gates.On,
		OpenKinds:    openKindsOf(rules),
	}), nil
}

// The kinds anyone may be drawn while the gates are on, in planet's words, as the page names a hand's kinds.
func openKindsOf(rules bonuses.Rules) []planetv1.BonusKind {
	open := make([]planetv1.BonusKind, 0, len(bonuses.Kinds))
	for _, kind := range bonuses.Kinds {
		if rules.Gates.Open[kind] {
			open = append(open, planetwords.Kind(kind))
		}
	}
	return open
}
