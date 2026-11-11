package sequencer

import (
	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

var Part = cprules.PartOf[*antibotv1.Rules, Rules]{
	Key: "sequencer", Read: func(m *antibotv1.Rules) Rules { return RulesOf(m.GetSequencer()) },
}

func RulesOf(m *antibotv1.SequencerRules) Rules {
	return Rules{
		Enabled:      m.GetEnabled(),
		MinSteps:     int(m.GetMinSteps()),
		MinShare:     m.GetMinShare(),
		CertainSteps: int(m.GetCertainSteps()),
		CertainShare: m.GetCertainShare(),
	}
}
