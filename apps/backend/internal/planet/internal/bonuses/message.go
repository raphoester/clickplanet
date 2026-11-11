package bonuses

import (
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

// RulesOf reads the offers' half of bonuses; planet's rule sets hold it as a part (rules/parts.Grants).
func RulesOf(m *planetv1.BonusRules) Rules {
	kinds, quiz := m.GetKinds(), m.GetQuiz()

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
		Bomb:         BombRules{Off: m.GetBomb().GetOff()},
		Enclose:      EncloseRules{MaxPerBox: int(m.GetEnclose().GetMaxPerBox())},
		Shield:       ShieldRules{MinPerBox: int(m.GetShield().GetMinPerBox()), MaxPerBox: int(m.GetShield().GetMaxPerBox())},
		Welcome:      WelcomeRules{Within: duration(m.GetWelcome().GetWithin()), BoxAfter: duration(m.GetWelcome().GetBoxAfter())},
		Quiz: QuizRules{
			Enabled:           quiz.GetEnabled(),
			MinInterval:       duration(quiz.GetMinInterval()),
			MaxInterval:       duration(quiz.GetMaxInterval()),
			MissRetry:         duration(quiz.GetMissRetry()),
			OfferTTL:          duration(quiz.GetOfferTtl()),
			AnswerWindow:      duration(quiz.GetAnswerWindow()),
			LeaderBias:        quiz.GetLeaderBias(),
			MaxChargesPerHour: int(quiz.GetMaxChargesPerHour()),
		},
		SignInRefill: m.GetSignInRefill(),
	}
}

func duration(d *durationpb.Duration) time.Duration {
	if d == nil {
		return 0
	}

	return d.AsDuration()
}
