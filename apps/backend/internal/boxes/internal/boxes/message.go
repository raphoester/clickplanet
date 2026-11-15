package boxes

import (
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	boxesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

// RulesOf reads a set's values: the boxes keep rule sets of their own (boxes.rule_sets).
func RulesOf(m *boxesv1.Rules) Rules {
	return Rules{
		Pace: Pace{
			MinInterval:     duration(m.GetMinInterval()),
			MaxInterval:     duration(m.GetMaxInterval()),
			MaxBoxesPerHour: int(m.GetMaxBoxesPerHour()),
		},
		OfferTTL:     duration(m.GetOfferTtl()),
		MissRetry:    duration(m.GetMissRetry()),
		ActiveWithin: duration(m.GetActiveWithin()),
		Welcome:      WelcomeRules{Within: duration(m.GetWelcome().GetWithin()), BoxAfter: duration(m.GetWelcome().GetBoxAfter())},
		Frozen:       m.GetFrozen(),
	}
}

func duration(d *durationpb.Duration) time.Duration {
	if d == nil {
		return 0
	}

	return d.AsDuration()
}

// Part is the boxes' rules in a set: every value of it.
var Part = cprules.PartOf[*boxesv1.Rules, Rules]{Key: "boxes", Read: RulesOf, Root: true}
