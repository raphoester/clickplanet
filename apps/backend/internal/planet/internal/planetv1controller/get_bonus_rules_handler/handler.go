// Package get_bonus_rules_handler serves planet.v1.ClickService/GetBonusRules.
package get_bonus_rules_handler

import (
	"context"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
)

// HomeSoil says whether native land takes two clicks. clicks.HomeSoil is one.
type HomeSoil interface {
	Enabled() bool
}

// New takes the rules themselves: they are fixed at boot, so there is nothing to ask a use case. The home-soil
// switch rides with the charges' sizes because the client reads both once, at load, to paint what the server
// will write.
func New(rules bonuses.Rules, homeSoil HomeSoil) GetBonusRulesHandler {
	return GetBonusRulesHandler{rules: rules, homeSoil: homeSoil}
}

type GetBonusRulesHandler struct {
	rules    bonuses.Rules
	homeSoil HomeSoil
}

func (h GetBonusRulesHandler) GetBonusRules(
	context.Context,
	*connect.Request[planetv1.GetBonusRulesRequest],
) (*connect.Response[planetv1.GetBonusRulesResponse], error) {
	return connect.NewResponse(&planetv1.GetBonusRulesResponse{
		BlastRadius:       h.rules.BlastRadius,
		EnclosureMaxTiles: uint32(h.rules.EnclosureMaxTiles),
		SpreadClicks:      uint32(h.rules.SpreadClicks),
		Enclosures:        uint32(h.rules.Enclosures),
		HomeSoil:          h.homeSoil.Enabled(),
	}), nil
}
