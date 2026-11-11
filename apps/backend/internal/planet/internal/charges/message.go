package charges

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

// Part reads the wallet's half of bonuses, and names its reasons bonuses.* as the set writes them.
var Part = rules.PartOf[Rules]{Key: "charges", Path: "bonuses", Read: func(m *planetv1.Rules) Rules {
	return RulesOf(m.GetBonuses())
}}

func RulesOf(m *planetv1.BonusRules) Rules {
	return Rules{
		Spread: SpreadRules{Clicks: int(m.GetSpread().GetClicks())},
		Bomb:   BombRules{Rings: m.GetBomb().GetRings(), Off: m.GetBomb().GetOff()},
		Enclose: EncloseRules{
			MaxTiles:  int(m.GetEnclose().GetMaxTiles()),
			Held:      int(m.GetEnclose().GetHeld()),
			Unlimited: m.GetEnclose().GetUnlimited(),
		},
		Shield: ShieldRules{Held: int(m.GetShield().GetHeld()), PerTile: int(m.GetShield().GetPerTile())},
	}
}
