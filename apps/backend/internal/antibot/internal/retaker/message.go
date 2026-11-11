package retaker

import (
	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

var Part = cprules.PartOf[*antibotv1.Rules, Rules]{
	Key: "retaker", Read: func(m *antibotv1.Rules) Rules { return RulesOf(m.GetRetaker()) },
}

func RulesOf(m *antibotv1.RetakerRules) Rules {
	return Rules{
		Enabled:        m.GetEnabled(),
		ReactionWindow: m.GetReactionWindow().AsDuration(),
		MinReactions:   int(m.GetMinReactions()),
		MaxSpread:      m.GetMaxSpread().AsDuration(),
		MaxMedian:      m.GetMaxMedian().AsDuration(),
		MinTiles:       int(m.GetMinTiles()),
		RoamMedian:     m.GetRoamMedian().AsDuration(),
		CertainTiles:   int(m.GetCertainTiles()),
	}
}
