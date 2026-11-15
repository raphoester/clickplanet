package sharing

import (
	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

var Part = cprules.PartOf[*antibotv1.Rules, Rules]{
	Key: "sharing", Read: func(m *antibotv1.Rules) Rules { return RulesOf(m.GetSharing()) },
}

func RulesOf(m *antibotv1.SharingRules) Rules {
	return Rules{
		Enabled:     m.GetEnabled(),
		Window:      m.GetWindow().AsDuration(),
		MaxAccounts: int(m.GetMaxAccounts()),
	}
}
