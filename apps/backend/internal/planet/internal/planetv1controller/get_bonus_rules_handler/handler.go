// Package get_bonus_rules_handler serves planet.v1.ClickService/GetBonusRules.
package get_bonus_rules_handler

import (
	"context"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
)

// New takes the rules themselves: they are fixed at boot, so there is nothing to ask a use case.
func New(rules bonuses.Rules) GetBonusRulesHandler {
	return GetBonusRulesHandler{rules: rules}
}

type GetBonusRulesHandler struct {
	rules bonuses.Rules
}

func (h GetBonusRulesHandler) GetBonusRules(
	context.Context,
	*connect.Request[planetv1.GetBonusRulesRequest],
) (*connect.Response[planetv1.GetBonusRulesResponse], error) {
	return connect.NewResponse(&planetv1.GetBonusRulesResponse{
		BlastRadius:       h.rules.BlastRadius,
		EnclosureMaxTiles: uint32(h.rules.EnclosureMaxTiles),
		SpreadClicks:      uint32(h.rules.SpreadClicks),
	}), nil
}
