package bonuses

import (
	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

// RulesOf reads a set's values: the bonuses module keeps rule sets of its own (bonuses.rule_sets).
func RulesOf(m *bonusesv1.Rules) Rules {
	kinds, open := m.GetKinds(), m.GetGates().GetOpen()

	return Rules{
		Kinds: map[Kind]float64{
			KindRefill:        kinds.GetRefill(),
			KindSpreadClicks:  kinds.GetSpreadClicks(),
			KindEncloseClicks: kinds.GetEncloseClicks(),
			KindBomb:          kinds.GetBomb(),
			KindShields:       kinds.GetShields(),
		},
		Gates: GateRules{
			On: m.GetGates().GetOn(),
			Open: map[Kind]bool{
				KindRefill:        open.GetRefill(),
				KindSpreadClicks:  open.GetSpreadClicks(),
				KindEncloseClicks: open.GetEncloseClicks(),
				KindBomb:          open.GetBomb(),
				KindShields:       open.GetShields(),
			},
		},
		Spread:       SpreadRules{MaxPerBox: int(m.GetSpread().GetMaxPerBox())},
		Enclose:      EncloseRules{MaxPerBox: int(m.GetEnclose().GetMaxPerBox())},
		Shield:       ShieldRules{MinPerBox: int(m.GetShield().GetMinPerBox()), MaxPerBox: int(m.GetShield().GetMaxPerBox())},
		SignInRefill: m.GetSignInRefill(),
	}
}

// Part is the bonuses module's rules in a set: every value of it.
var Part = cprules.PartOf[*bonusesv1.Rules, Rules]{Key: "offers", Read: RulesOf}
