package bonuses

import (
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

var Part = rules.PartOf[Rules]{Key: "bonuses", Read: func(m *planetv1.Rules) Rules { return RulesOf(m.GetBonuses()) }}

func RulesOf(m *planetv1.BonusRules) Rules {
	kinds, quiz, ad := m.GetKinds(), m.GetQuiz(), m.GetAdRefill()

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
		Spread:       SpreadRules{Clicks: int(m.GetSpread().GetClicks()), MaxPerBox: int(m.GetSpread().GetMaxPerBox())},
		Bomb:         BombRules{Rings: m.GetBomb().GetRings()},
		Enclose: EncloseRules{
			MaxTiles:  int(m.GetEnclose().GetMaxTiles()),
			Held:      int(m.GetEnclose().GetHeld()),
			MaxPerBox: int(m.GetEnclose().GetMaxPerBox()),
		},
		Shield: ShieldRules{
			Held:      int(m.GetShield().GetHeld()),
			MinPerBox: int(m.GetShield().GetMinPerBox()),
			MaxPerBox: int(m.GetShield().GetMaxPerBox()),
			PerTile:   int(m.GetShield().GetPerTile()),
		},
		Welcome: WelcomeRules{Within: duration(m.GetWelcome().GetWithin()), BoxAfter: duration(m.GetWelcome().GetBoxAfter())},
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
		AdRefill:     AdRefillRules{Enabled: ad.GetEnabled(), Every: duration(ad.GetEvery())},
	}
}

func duration(d *durationpb.Duration) time.Duration {
	if d == nil {
		return 0
	}

	return d.AsDuration()
}
