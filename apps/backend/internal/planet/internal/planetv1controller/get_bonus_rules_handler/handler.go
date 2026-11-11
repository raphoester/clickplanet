package get_bonus_rules_handler

import (
	"context"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Toll interface {
	Steps() []clicks.TollStep
}

type Blast interface {
	Radius() float64
}

// New answers the wallet's half of the bonus rules, and of the offers' half only signInRefill, which tells the client
// whether a sign-in fills the bank. In B it is GetChargeRules, and signInRefill is the offers' to answer.
func New(
	rules func() charges.Rules, signInRefill func() bool, blast Blast, toll Toll, reach func() clicks.Reaching,
	capitals func() clicks.CapitalRules,
) GetBonusRulesHandler {
	return GetBonusRulesHandler{
		rules: rules, signInRefill: signInRefill, blast: blast, toll: toll, reach: reach, capitals: capitals,
	}
}

type GetBonusRulesHandler struct {
	rules        func() charges.Rules
	signInRefill func() bool
	blast        Blast
	toll         Toll
	reach        func() clicks.Reaching
	capitals     func() clicks.CapitalRules
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
		SignInRefill:      h.signInRefill(),

		EnclosuresUnlimited: rules.Enclose.Unlimited,

		Reach:               bool(h.reach()),
		KnockOut:            capitals.KnockOut,
		CapitalShieldsFixed: capitals.FixedShields,
		BombsOff:            rules.Bomb.Off,
	}), nil
}

func tollStepsOf(steps []clicks.TollStep) []*planetv1.TollStep {
	out := make([]*planetv1.TollStep, len(steps))
	for i, step := range steps {
		out[i] = &planetv1.TollStep{Share: step.Share, Slowdown: step.Slowdown}
	}

	return out
}
