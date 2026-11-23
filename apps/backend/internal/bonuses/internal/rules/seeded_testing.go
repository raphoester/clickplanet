//go:build testing

package rules

import (
	"google.golang.org/protobuf/proto"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

// Seeded is the set the migrations keep as default on a database with no planet rules to copy: production's default of
// 11 November 2026, the offers' half of planet's, with the gates off, and no virus, no mine, no rally and no trap. The quiz's values it held are the
// quizzes module's since, and the boxes' the boxes module's: read no more.
func Seeded() cprules.Snapshot[*bonusesv1.Rules] {
	f, n, on := proto.Float64, proto.Uint32, proto.Bool
	return cprules.Snapshot[*bonusesv1.Rules]{
		Name: cprules.Default,
		Rules: &bonusesv1.Rules{
			Kinds: &bonusesv1.KindWeights{
				Refill: f(5), SpreadClicks: f(3), EncloseClicks: f(2), Bomb: f(1), Shields: f(3), Virus: f(0), Mine: f(0), Rally: f(0),
				Trap: f(0),
			},
			Gates: &bonusesv1.GateRules{On: on(false), Open: &bonusesv1.OpenKinds{
				Refill: on(true), SpreadClicks: on(true), EncloseClicks: on(false), Bomb: on(false), Shields: on(true), Virus: on(false),
				Mine: on(false), Rally: on(false), Trap: on(false),
			}},
			Spread:       &bonusesv1.SpreadRules{MaxPerBox: n(4)},
			Enclose:      &bonusesv1.EncloseRules{MaxPerBox: n(3)},
			Shield:       &bonusesv1.ShieldRules{MinPerBox: n(5), MaxPerBox: n(20)},
			Mine:         &bonusesv1.MineRules{MaxPerBox: n(2)},
			Rally:        &bonusesv1.RallyRules{MaxPerBox: n(1)},
			Trap:         &bonusesv1.TrapRules{MaxPerBox: n(2)},
			SignInRefill: on(true),
		},
	}
}

// Named is default's values under another name, changed: a set that holds every value.
func Named(name cprules.SetName, change func(*bonusesv1.Rules)) cprules.Snapshot[*bonusesv1.Rules] {
	kept := Seeded()
	kept.Name = name
	change(kept.Rules)
	return kept
}

func AsIs(*bonusesv1.Rules) {}
