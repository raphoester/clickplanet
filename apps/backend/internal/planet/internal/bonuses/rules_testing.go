//go:build testing

package bonuses

import "time"

// TestRules are production's rules of October 2026, with no welcome and no quiz.
func TestRules() Rules {
	return Rules{
		Boxes:        Pace{MinInterval: 4 * time.Minute, MaxInterval: 8 * time.Minute, MaxChargesPerHour: 12},
		Kinds:        map[Kind]float64{KindRefill: 5, KindSpreadClicks: 3, KindEncloseClicks: 2, KindBomb: 1, KindShields: 3},
		OfferTTL:     15 * time.Second,
		MissRetry:    2 * time.Minute,
		ActiveWithin: 2 * time.Minute,
		Spread:       SpreadRules{Clicks: 8, MaxPerBox: 4},
		Bomb:         BombRules{Rings: 4},
		Enclose:      EncloseRules{MaxTiles: 25, Held: 3, MaxPerBox: 3},
		Shield:       ShieldRules{Held: 30, MinPerBox: 5, MaxPerBox: 20, PerTile: 10},
		Quiz: QuizRules{
			MinInterval: 4 * time.Minute, MaxInterval: 8 * time.Minute, MissRetry: 2 * time.Minute,
			OfferTTL: 25 * time.Second, AnswerWindow: 8 * time.Second, LeaderBias: 1, MaxChargesPerHour: 12,
		},
	}
}
