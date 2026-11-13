//go:build testing

package bonuses

import "time"

// TestRules are production's grant rules of October 2026, with no welcome.
func TestRules() Rules {
	return Rules{
		Boxes:        Pace{MinInterval: 4 * time.Minute, MaxInterval: 8 * time.Minute, MaxChargesPerHour: 12},
		Kinds:        map[Kind]float64{KindRefill: 5, KindSpreadClicks: 3, KindEncloseClicks: 2, KindBomb: 1, KindShields: 3},
		OfferTTL:     15 * time.Second,
		MissRetry:    2 * time.Minute,
		ActiveWithin: 2 * time.Minute,
		Spread:       SpreadRules{MaxPerBox: 4},
		Enclose:      EncloseRules{MaxPerBox: 3},
		Shield:       ShieldRules{MinPerBox: 5, MaxPerBox: 20},
	}
}
