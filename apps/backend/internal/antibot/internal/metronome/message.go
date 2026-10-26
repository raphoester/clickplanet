package metronome

import (
	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
)

var Part = rules.PartOf[Rules]{Key: "metronome", Read: func(m *antibotv1.Rules) Rules { return RulesOf(m.GetMetronome()) }}

func RulesOf(m *antibotv1.MetronomeRules) Rules {
	shape, clock, stamina := m.GetShape(), m.GetClock(), m.GetStamina()

	var maxSkew, certainSkew, minCoherence, certainCoherence *float64
	if shape != nil {
		maxSkew, certainSkew = copied(shape.MaxSkew), copied(shape.CertainSkew)
	}
	if clock != nil {
		minCoherence, certainCoherence = copied(clock.MinCoherence), copied(clock.CertainCoherence)
	}

	return Rules{
		Enabled:       m.GetEnabled(),
		MaxGap:        m.GetMaxGap().AsDuration(),
		MaxSpread:     m.GetMaxSpread().AsDuration(),
		MinClicks:     int(m.GetMinClicks()),
		CertainFor:    m.GetCertainFor().AsDuration(),
		CertainClicks: int(m.GetCertainClicks()),
		Shape: ShapeRules{
			MaxGap:        shape.GetMaxGap().AsDuration(),
			Clicks:        int(shape.GetClicks()),
			MaxSkew:       maxSkew,
			CertainClicks: int(shape.GetCertainClicks()),
			CertainSkew:   certainSkew,
		},
		Clock: ClockRules{
			Period:           clock.GetPeriod().AsDuration(),
			Clicks:           int(clock.GetClicks()),
			MinCoherence:     minCoherence,
			CertainClicks:    int(clock.GetCertainClicks()),
			CertainFor:       clock.GetCertainFor().AsDuration(),
			CertainCoherence: certainCoherence,
		},
		Stamina: StaminaRules{
			Clicks:      int(stamina.GetClicks()),
			Window:      stamina.GetWindow().AsDuration(),
			MinBusy:     stamina.GetMinBusy().AsDuration(),
			CertainBusy: stamina.GetCertainBusy().AsDuration(),
		},
	}
}

func copied(value *float64) *float64 {
	if value == nil {
		return nil
	}
	kept := *value
	return &kept
}
