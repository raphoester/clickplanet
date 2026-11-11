package charges

import (
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

// Rules is the half of planet.v1.BonusRules the wallet reads: how many of each charge a hand holds, and what each does
// when it is spent. What a box or a quiz grants is the other half, the offers' (bonuses.Rules).
type Rules struct {
	Spread  SpreadRules
	Bomb    BombRules
	Enclose EncloseRules
	Shield  ShieldRules
}

// A tile keeps its shields in a byte.
const mostShieldsOnATile = 255

func (r Rules) Validate() error {
	return errors.Join(
		rules.Positive("spread.clicks", r.Spread.Clicks),
		rules.Positive("bomb.rings", r.Bomb.Rings), rules.Finite("bomb.rings", r.Bomb.Rings),
		rules.Positive("enclose.maxTiles", r.Enclose.MaxTiles),
		rules.Positive("enclose.held", r.Enclose.Held),
		rules.Positive("shield.held", r.Shield.Held),
		rules.Positive("shield.perTile", r.Shield.PerTile),
		rules.AtMost("shield.perTile", r.Shield.PerTile, "what a tile keeps", mostShieldsOnATile),
	)
}

type SpreadRules struct {
	Clicks int
}

// BombRules: Off, a bomb held is not dropped. The offers read bomb.off too, and offer none.
type BombRules struct {
	Rings float64
	Off   bool
}

type EncloseRules struct {
	MaxTiles  int
	Held      int
	Unlimited bool
}

type ShieldRules struct {
	Held    int
	PerTile int
}
