package cohort

import (
	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
)

var Part = rules.PartOf[Rules]{Key: "cohort", Read: func(m *antibotv1.Rules) Rules { return RulesOf(m.GetCohort()) }}

func RulesOf(m *antibotv1.CohortRules) Rules {
	return Rules{
		Enabled:        m.GetEnabled(),
		StartWindow:    m.GetStartWindow().AsDuration(),
		MinClicks:      int(m.GetMinClicks()),
		MinFlagShare:   m.GetMinFlagShare(),
		RateRatio:      m.GetRateRatio(),
		LengthRatio:    m.GetLengthRatio(),
		QuietAfter:     m.GetQuietAfter().AsDuration(),
		MinMembers:     int(m.GetMinMembers()),
		CertainCohorts: int(m.GetCertainCohorts()),
		CertainMembers: int(m.GetCertainMembers()),
		ChainWindow:    m.GetChainWindow().AsDuration(),
	}
}
