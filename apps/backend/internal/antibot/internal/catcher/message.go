package catcher

import (
	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
)

var Part = rules.PartOf[Rules]{Key: "catcher", Read: func(m *antibotv1.Rules) Rules { return RulesOf(m.GetCatcher()) }}

func RulesOf(m *antibotv1.CatcherRules) Rules {
	return Rules{
		Enabled:       m.GetEnabled(),
		MinCatches:    int(m.GetMinCatches()),
		MaxMedian:     m.GetMaxMedian().AsDuration(),
		CertainMedian: m.GetCertainMedian().AsDuration(),
		Foreign: ForeignRules{
			Window:        m.GetForeign().GetWindow().AsDuration(),
			MinClaims:     int(m.GetForeign().GetMinClaims()),
			CertainClaims: int(m.GetForeign().GetCertainClaims()),
		},
	}
}
