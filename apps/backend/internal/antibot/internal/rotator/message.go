package rotator

import (
	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
)

var Part = rules.PartOf[Rules]{Key: "rotator", Read: func(m *antibotv1.Rules) Rules { return RulesOf(m.GetRotator()) }}

func RulesOf(m *antibotv1.RotatorRules) Rules {
	return Rules{
		Enabled:      m.GetEnabled(),
		Window:       m.GetWindow().AsDuration(),
		MinSeconds:   int(m.GetMinSeconds()),
		MinByChance:  m.GetMinByChance(),
		MaxShare:     m.GetMaxShare(),
		MinSlots:     int(m.GetMinSlots()),
		CertainSlots: int(m.GetCertainSlots()),
	}
}
