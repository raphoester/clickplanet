package quizzes

import (
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/bank"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

// RulesOf reads a set's values: the quizzes keep rule sets of their own (quizzes.rule_sets).
func RulesOf(m *quizzesv1.Rules) Rules {
	return Rules{
		Enabled:                m.GetEnabled(),
		MinInterval:            duration(m.GetMinInterval()),
		MaxInterval:            duration(m.GetMaxInterval()),
		MissRetry:              duration(m.GetMissRetry()),
		OfferTTL:               duration(m.GetOfferTtl()),
		AnswerWindow:           duration(m.GetAnswerWindow()),
		LeaderBias:             m.GetLeaderBias(),
		MaxRightAnswersPerHour: int(m.GetMaxRightAnswersPerHour()),
		Fit: bank.Fit{
			Target:     m.GetTarget(),
			Width:      m.GetWidth(),
			Floor:      m.GetFloor(),
			FloorAfter: int(m.GetFloorAfter()),
			Explore:    m.GetExplore(),
		},
		ActiveWithin: duration(m.GetActiveWithin()),
		Welcome:      WelcomeRules{Within: duration(m.GetWelcome().GetWithin())},
		Frozen:       m.GetFrozen(),
	}
}

func duration(d *durationpb.Duration) time.Duration {
	if d == nil {
		return 0
	}

	return d.AsDuration()
}

// Part is the quizzes' rules in a set: every value of it, each named by its path in quizzes.v1.Rules.
var Part = cprules.PartOf[*quizzesv1.Rules, Rules]{Key: "quizzes", Read: RulesOf, Root: true}
