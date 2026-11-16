//go:build testing

package bonuses

// TestRules are production's grant rules of October 2026.
func TestRules() Rules {
	return Rules{
		Kinds:   map[Kind]float64{KindRefill: 5, KindSpreadClicks: 3, KindEncloseClicks: 2, KindBomb: 1, KindShields: 3, KindVirus: 0},
		Spread:  SpreadRules{MaxPerBox: 4},
		Enclose: EncloseRules{MaxPerBox: 3},
		Shield:  ShieldRules{MinPerBox: 5, MaxPerBox: 20},
	}
}
