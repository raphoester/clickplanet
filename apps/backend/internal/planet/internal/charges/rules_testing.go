//go:build testing

package charges

// TestRules are production's charge rules of October 2026.
func TestRules() Rules {
	return Rules{
		Spread:  SpreadRules{Clicks: 8},
		Bomb:    BombRules{Rings: 4},
		Enclose: EncloseRules{MaxTiles: 25, Held: 3},
		Shield:  ShieldRules{Held: 30, PerTile: 10},
	}
}
