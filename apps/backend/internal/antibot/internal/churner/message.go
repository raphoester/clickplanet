package churner

import (
	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
)

var Part = rules.PartOf[Rules]{Key: "churner", Read: func(m *antibotv1.Rules) Rules { return RulesOf(m.GetChurner()) }}

func RulesOf(m *antibotv1.ChurnerRules) Rules {
	return Rules{
		Enabled: m.GetEnabled(),
		Window:  m.GetWindow().AsDuration(),
		V6: Bounds{
			MinAccounts:     int(m.GetV6().GetMinAccounts()),
			CertainAccounts: int(m.GetV6().GetCertainAccounts()),
		},
		V4: Bounds{
			MinAccounts:     int(m.GetV4().GetMinAccounts()),
			CertainAccounts: int(m.GetV4().GetCertainAccounts()),
		},
		Relay: RelayRules{
			Handoff:      m.GetRelay().GetHandoff().AsDuration(),
			MaxLife:      m.GetRelay().GetMaxLife().AsDuration(),
			MinClicks:    int(m.GetRelay().GetMinClicks()),
			MinFlagShare: m.GetRelay().GetMinFlagShare(),
			MinLinks:     int(m.GetRelay().GetMinLinks()),
			CertainLinks: int(m.GetRelay().GetCertainLinks()),
		},
	}
}
