//go:build testing

package charges

import "time"

// TestRules are production's charge rules of October 2026.
func TestRules() Rules {
	return Rules{
		Spread:  SpreadRules{Clicks: 8, Rings: 1},
		Bomb:    BombRules{Rings: 4},
		Enclose: EncloseRules{MaxTiles: 25, Held: 3},
		Shield:  ShieldRules{Held: 30, PerTile: 10},
		Virus:   VirusRules{Infections: 32, Every: 20 * time.Second},
	}
}
