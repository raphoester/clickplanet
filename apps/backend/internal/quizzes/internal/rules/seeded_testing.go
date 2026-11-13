//go:build testing

package rules

import (
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

// Seeded is the set the migrations keep as default on a database with no bonuses rules to copy: production's quiz half
// of 13 November 2026, the difficulty-weighted draw on.
func Seeded() cprules.Snapshot[*quizzesv1.Rules] {
	d := durationpb.New
	f, n, on := proto.Float64, proto.Uint32, proto.Bool
	return cprules.Snapshot[*quizzesv1.Rules]{
		Name: cprules.Default,
		Rules: &quizzesv1.Rules{
			Enabled:                on(true),
			MinInterval:            d(4 * time.Minute),
			MaxInterval:            d(8 * time.Minute),
			MissRetry:              d(2 * time.Minute),
			OfferTtl:               d(25 * time.Second),
			AnswerWindow:           d(8 * time.Second),
			LeaderBias:             f(1),
			MaxRightAnswersPerHour: n(12),
			Target:                 f(0.7),
			Width:                  f(1),
			Floor:                  f(0.42),
			FloorAfter:             n(15),
			Explore:                f(0.1),
			ActiveWithin:           d(2 * time.Minute),
			Welcome:                &quizzesv1.WelcomeRules{Within: d(10 * time.Minute)},
			Frozen:                 on(false),
		},
	}
}

// Named is default's values under another name, changed: a set that holds every value.
func Named(name cprules.SetName, change func(*quizzesv1.Rules)) cprules.Snapshot[*quizzesv1.Rules] {
	kept := Seeded()
	kept.Name = name
	change(kept.Rules)
	return kept
}

func AsIs(*quizzesv1.Rules) {}
