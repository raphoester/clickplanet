package get_bonus_rules_handler

import (
	"context"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type HomeSoil interface {
	Enabled() bool
}

type Toll interface {
	Steps() []clicks.TollStep
}

func New(rules bonuses.Rules, homeSoil HomeSoil, toll Toll) GetBonusRulesHandler {
	return GetBonusRulesHandler{rules: rules, homeSoil: homeSoil, toll: toll}
}

type GetBonusRulesHandler struct {
	rules    bonuses.Rules
	homeSoil HomeSoil
	toll     Toll
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
		TollSteps:         tollStepsOf(h.toll.Steps()),
	}), nil
}

func tollStepsOf(steps []clicks.TollStep) []*planetv1.TollStep {
	out := make([]*planetv1.TollStep, len(steps))
	for i, step := range steps {
		out[i] = &planetv1.TollStep{Share: step.Share, Slowdown: step.Slowdown}
	}

	return out
}
