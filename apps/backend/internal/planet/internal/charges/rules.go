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

const MostSpreadRings = 3

func (r Rules) Validate() error {
	return errors.Join(
		rules.Positive("spread.clicks", r.Spread.Clicks),
		rules.Positive("spread.rings", r.Spread.Rings),
		rules.AtMost("spread.rings", r.Spread.Rings, "the widest spread", MostSpreadRings),
		rules.Positive("bomb.rings", r.Bomb.Rings), rules.Finite("bomb.rings", r.Bomb.Rings),
		rules.Positive("enclose.maxTiles", r.Enclose.MaxTiles),
		rules.Positive("enclose.held", r.Enclose.Held),
		rules.Positive("shield.held", r.Shield.Held),
		rules.Positive("shield.perTile", r.Shield.PerTile),
		rules.AtMost("shield.perTile", r.Shield.PerTile, "what a tile keeps", mostShieldsOnATile),
		r.Enclose.validate(),
	)
}

type SpreadRules struct {
	Clicks    int
	Rings     int
	Unlimited bool
}

// BombRules: Off, a bomb held is not dropped, and no hand has room for one, so none is granted.
type BombRules struct {
	Rings float64
	Off   bool
}

// EncloseRules: Off, a click with enclose on closes nothing and spends nothing, and no hand has room for one, so none
// is granted. The enclosures held are kept.
type EncloseRules struct {
	MaxTiles  int
	Held      int
	Unlimited bool
	Off       bool
}

func (r EncloseRules) validate() error {
	if r.Off && r.Unlimited {
		return errors.New("enclose.off is on with enclose.unlimited: an enclosure cannot be off and free")
	}
	return nil
}

type ShieldRules struct {
	Held    int
	PerTile int
}
