package defender

import (
	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
)

var Part = rules.PartOf[Rules]{Key: "defender", Read: func(m *antibotv1.Rules) Rules { return RulesOf(m.GetDefender()) }}

func RulesOf(m *antibotv1.DefenderRules) Rules {
	return Rules{
		Enabled:       m.GetEnabled(),
		RetakeWindow:  m.GetRetakeWindow().AsDuration(),
		MinClicks:     int(m.GetMinClicks()),
		MinShare:      m.GetMinShare(),
		CertainClicks: int(m.GetCertainClicks()),
		CertainShare:  m.GetCertainShare(),
	}
}
