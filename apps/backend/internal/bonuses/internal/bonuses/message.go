package bonuses

import (
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

// RulesOf reads a set's values: the offers keep rule sets of their own (bonuses.rule_sets).
func RulesOf(m *bonusesv1.Rules) Rules {
	kinds := m.GetKinds()

	return Rules{
		Boxes: Pace{
			MinInterval:       duration(m.GetBoxes().GetMinInterval()),
			MaxInterval:       duration(m.GetBoxes().GetMaxInterval()),
			MaxChargesPerHour: int(m.GetBoxes().GetMaxChargesPerHour()),
		},
		Kinds: map[Kind]float64{
			KindRefill:        kinds.GetRefill(),
			KindSpreadClicks:  kinds.GetSpreadClicks(),
			KindEncloseClicks: kinds.GetEncloseClicks(),
			KindBomb:          kinds.GetBomb(),
			KindShields:       kinds.GetShields(),
		},
		OfferTTL:     duration(m.GetOfferTtl()),
		MissRetry:    duration(m.GetMissRetry()),
		ActiveWithin: duration(m.GetActiveWithin()),
		Spread:       SpreadRules{MaxPerBox: int(m.GetSpread().GetMaxPerBox())},
		Enclose:      EncloseRules{MaxPerBox: int(m.GetEnclose().GetMaxPerBox())},
		Shield:       ShieldRules{MinPerBox: int(m.GetShield().GetMinPerBox()), MaxPerBox: int(m.GetShield().GetMaxPerBox())},
		Welcome:      WelcomeRules{Within: duration(m.GetWelcome().GetWithin()), BoxAfter: duration(m.GetWelcome().GetBoxAfter())},
		SignInRefill: m.GetSignInRefill(),
		Frozen:       m.GetFrozen(),
	}
}

func duration(d *durationpb.Duration) time.Duration {
	if d == nil {
		return 0
	}

	return d.AsDuration()
}

// Part is the offers' rules in a set: every value of it.
var Part = cprules.PartOf[*bonusesv1.Rules, Rules]{Key: "offers", Read: RulesOf}
