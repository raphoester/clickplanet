package get_charge_rules_handler

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

// New answers the wallet's rules, how much a hand holds and what each charge does, and the knobs of the map the page
// shows: the toll, the reach and the capitals.
func New(
	rules func() charges.Rules, blast Blast, toll Toll, reach func() clicks.Reaching, capitals func() clicks.CapitalRules,
) GetChargeRulesHandler {
	return GetChargeRulesHandler{rules: rules, blast: blast, toll: toll, reach: reach, capitals: capitals}
}

type GetChargeRulesHandler struct {
	rules    func() charges.Rules
	blast    Blast
	toll     Toll
	reach    func() clicks.Reaching
	capitals func() clicks.CapitalRules
}

func (h GetChargeRulesHandler) GetChargeRules(
	context.Context,
	*connect.Request[planetv1.GetChargeRulesRequest],
) (*connect.Response[planetv1.GetChargeRulesResponse], error) {
	rules, capitals := h.rules(), h.capitals()

	return connect.NewResponse(&planetv1.GetChargeRulesResponse{
		BlastRadius:       h.blast.Radius(),
		EnclosureMaxTiles: uint32(rules.Enclose.MaxTiles),
		SpreadClicks:      uint32(rules.Spread.Clicks),
		Enclosures:        uint32(rules.Enclose.Held),
		TollSteps:         tollStepsOf(h.toll.Steps()),
		Shields:           uint32(rules.Shield.Held),
		TileShields:       uint32(rules.Shield.PerTile),

		EnclosuresUnlimited: rules.Enclose.Unlimited,

		Reach:               bool(h.reach()),
		KnockOut:            capitals.KnockOut,
		CapitalShieldsFixed: capitals.FixedShields,
		BombsOff:            rules.Bomb.Off,

		SpreadRings:     uint32(rules.Spread.Rings),
		SpreadUnlimited: rules.Spread.Unlimited,
	}), nil
}

func tollStepsOf(steps []clicks.TollStep) []*planetv1.TollStep {
	out := make([]*planetv1.TollStep, len(steps))
	for i, step := range steps {
		out[i] = &planetv1.TollStep{Share: step.Share, Slowdown: step.Slowdown}
	}

	return out
}
