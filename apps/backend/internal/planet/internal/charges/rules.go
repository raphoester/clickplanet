package charges

import (
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

// Rules is what the wallet reads (planet.v1.ChargeRules): how many of each charge a hand holds, and what each does when
// it is spent. What a box or a quiz grants is not planet's: whoever grants is told what a hand has no room for.
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

// BombRules: Off, a bomb held is not dropped, and no hand has room for one, so none is granted.
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
