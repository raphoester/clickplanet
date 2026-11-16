//go:build testing

package bonuses

// TestRules are production's grant rules of October 2026, with the gates off, as every set holds them until the levels
// are shown.
func TestRules() Rules {
	return Rules{
		Kinds:   map[Kind]float64{KindRefill: 5, KindSpreadClicks: 3, KindEncloseClicks: 2, KindBomb: 1, KindShields: 3},
		Gates:   GateRules{Open: map[Kind]bool{KindRefill: true, KindSpreadClicks: true, KindShields: true}},
		Spread:  SpreadRules{MaxPerBox: 4},
		Enclose: EncloseRules{MaxPerBox: 3},
		Shield:  ShieldRules{MinPerBox: 5, MaxPerBox: 20},
	}
}

// TestGated is TestRules with the gates on: anyone may be drawn a refill, a spread or shields, and the rest once
// progression unlocks it.
func TestGated() Rules {
	rules := TestRules()
	rules.Gates.On = true
	return rules
}
