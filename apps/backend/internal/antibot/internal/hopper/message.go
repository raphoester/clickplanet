package hopper

import (
	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

var Part = cprules.PartOf[*antibotv1.Rules, Rules]{
	Key: "hopper", Read: func(m *antibotv1.Rules) Rules { return RulesOf(m.GetHopper()) },
}

func RulesOf(m *antibotv1.HopperRules) Rules {
	return Rules{
		Enabled:      m.GetEnabled(),
		MinAngle:     m.GetMinAngle(),
		MinGap:       m.GetMinGap().AsDuration(),
		MaxGap:       m.GetMaxGap().AsDuration(),
		MinSteps:     int(m.GetMinSteps()),
		MinShare:     m.GetMinShare(),
		CertainSteps: int(m.GetCertainSteps()),
		CertainShare: m.GetCertainShare(),
	}
}
