//go:build testing

package rules

import (
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

// Seeded is the set the first migration keeps as default on a database with no planet rules to copy: production's
// default of 11 November 2026, the offers' half of planet's.
func Seeded() cprules.Snapshot[*bonusesv1.Rules] {
	d := durationpb.New
	f, n, on := proto.Float64, proto.Uint32, proto.Bool
	return cprules.Snapshot[*bonusesv1.Rules]{
		Name: cprules.Default,
		Rules: &bonusesv1.Rules{
			Boxes:        &bonusesv1.BoxPace{MinInterval: d(4 * time.Minute), MaxInterval: d(8 * time.Minute), MaxChargesPerHour: n(12)},
			Kinds:        &bonusesv1.KindWeights{Refill: f(5), SpreadClicks: f(3), EncloseClicks: f(2), Bomb: f(1), Shields: f(3)},
			OfferTtl:     d(15 * time.Second),
			MissRetry:    d(2 * time.Minute),
			ActiveWithin: d(2 * time.Minute),
			Spread:       &bonusesv1.SpreadRules{MaxPerBox: n(4)},
			Enclose:      &bonusesv1.EncloseRules{MaxPerBox: n(3)},
			Shield:       &bonusesv1.ShieldRules{MinPerBox: n(5), MaxPerBox: n(20)},
			Welcome:      &bonusesv1.WelcomeRules{Within: d(10 * time.Minute), BoxAfter: d(time.Minute)},
			Quiz: &bonusesv1.QuizRules{
				Enabled: on(true), MinInterval: d(4 * time.Minute), MaxInterval: d(8 * time.Minute),
				MissRetry: d(2 * time.Minute), OfferTtl: d(25 * time.Second), AnswerWindow: d(8 * time.Second),
				LeaderBias: f(1), MaxChargesPerHour: n(12),
			},
			SignInRefill: on(true),
			Frozen:       on(false),
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
