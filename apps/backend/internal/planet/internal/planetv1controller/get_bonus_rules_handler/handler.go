package get_bonus_rules_handler

import (
	"context"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Toll interface {
	Steps() []clicks.TollStep
}

type Blast interface {
	Radius() float64
}

func New(rules bonuses.InForce, blast Blast, toll Toll) GetBonusRulesHandler {
	return GetBonusRulesHandler{rules: rules, blast: blast, toll: toll}
}

type GetBonusRulesHandler struct {
	rules bonuses.InForce
	blast Blast
	toll  Toll
}

func (h GetBonusRulesHandler) GetBonusRules(
	context.Context,
	*connect.Request[planetv1.GetBonusRulesRequest],
) (*connect.Response[planetv1.GetBonusRulesResponse], error) {
	rules := h.rules.Bonuses()

	return connect.NewResponse(&planetv1.GetBonusRulesResponse{
		BlastRadius:       h.blast.Radius(),
		EnclosureMaxTiles: uint32(rules.Enclose.MaxTiles),
		SpreadClicks:      uint32(rules.Spread.Clicks),
		Enclosures:        uint32(rules.Enclose.Held),
		TollSteps:         tollStepsOf(h.toll.Steps()),
		Shields:           uint32(rules.Shield.Held),
		TileShields:       uint32(rules.Shield.PerTile),
	}), nil
}

func tollStepsOf(steps []clicks.TollStep) []*planetv1.TollStep {
	out := make([]*planetv1.TollStep, len(steps))
	for i, step := range steps {
		out[i] = &planetv1.TollStep{Share: step.Share, Slowdown: step.Slowdown}
	}

	return out
}
