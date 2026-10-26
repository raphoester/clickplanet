package jury

import (
	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
)

var Part = rules.PartOf[Rules]{Key: "jury", Read: func(m *antibotv1.Rules) Rules { return RulesOf(m.GetJury()) }}

func RulesOf(m *antibotv1.JuryRules) Rules {
	return Rules{
		MinSuspects:     int(m.GetMinSuspects()),
		SuspicionWindow: m.GetSuspicionWindow().AsDuration(),
	}
}
