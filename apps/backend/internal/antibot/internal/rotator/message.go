package rotator

import (
	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

var Part = cprules.PartOf[*antibotv1.Rules, Rules]{
	Key: "rotator", Read: func(m *antibotv1.Rules) Rules { return RulesOf(m.GetRotator()) },
}

func RulesOf(m *antibotv1.RotatorRules) Rules {
	return Rules{
		Enabled:         m.GetEnabled(),
		Window:          m.GetWindow().AsDuration(),
		MinSeconds:      int(m.GetMinSeconds()),
		MinByChance:     m.GetMinByChance(),
		MaxShare:        m.GetMaxShare(),
		MinSlots:        int(m.GetMinSlots()),
		CertainSlots:    int(m.GetCertainSlots()),
		CertainByChance: m.GetCertainByChance(),
	}
}
