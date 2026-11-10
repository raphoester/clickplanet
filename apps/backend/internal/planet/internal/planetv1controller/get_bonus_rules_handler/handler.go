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

func New(
	rules func() bonuses.Rules, blast Blast, toll Toll, reach func() clicks.Reaching, capitals func() clicks.CapitalRules,
) GetBonusRulesHandler {
	return GetBonusRulesHandler{rules: rules, blast: blast, toll: toll, reach: reach, capitals: capitals}
}

type GetBonusRulesHandler struct {
	rules    func() bonuses.Rules
	blast    Blast
	toll     Toll
	reach    func() clicks.Reaching
	capitals func() clicks.CapitalRules
}

func (h GetBonusRulesHandler) GetBonusRules(
	context.Context,
	*connect.Request[planetv1.GetBonusRulesRequest],
) (*connect.Response[planetv1.GetBonusRulesResponse], error) {
	rules, capitals := h.rules(), h.capitals()

	return connect.NewResponse(&planetv1.GetBonusRulesResponse{
		BlastRadius:       h.blast.Radius(),
		EnclosureMaxTiles: uint32(rules.Enclose.MaxTiles),
		SpreadClicks:      uint32(rules.Spread.Clicks),
		Enclosures:        uint32(rules.Enclose.Held),
		TollSteps:         tollStepsOf(h.toll.Steps()),
		Shields:           uint32(rules.Shield.Held),
		TileShields:       uint32(rules.Shield.PerTile),
		SignInRefill:      rules.SignInRefill,

		EnclosuresUnlimited: rules.Enclose.Unlimited,

		Reach:               bool(h.reach()),
		KnockOut:            capitals.KnockOut,
		CapitalShieldsFixed: capitals.FixedShields,
	}), nil
}

func tollStepsOf(steps []clicks.TollStep) []*planetv1.TollStep {
	out := make([]*planetv1.TollStep, len(steps))
	for i, step := range steps {
		out[i] = &planetv1.TollStep{Share: step.Share, Slowdown: step.Slowdown}
	}

	return out
}
