package charges

import (
	"errors"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

// Rules is what the wallet reads (planet.v1.ChargeRules): how many of each charge a hand holds, and what each does when
// it is spent. What a box or a quiz grants is not planet's: whoever grants draws blind, and the wallet keeps what fits.
type Rules struct {
	Refill  RefillRules
	Spread  SpreadRules
	Bomb    BombRules
	Enclose EncloseRules
	Shield  ShieldRules
	Virus   VirusRules
	Mine    MineRules
	Rally   RallyRules
}

// Size is how many of the kind a hand holds at most.
func (r Rules) Size(kind Kind) int {
	switch kind {
	case KindRefill:
		return r.Refill.Held
	case KindBomb:
		return r.Bomb.Held
	case KindEncloseClicks:
		return r.Enclose.Held
	case KindSpreadClicks:
		return r.Spread.Clicks
	case KindShields:
		return r.Shield.Held
	case KindVirus:
		return r.Virus.Held
	case KindMine:
		return r.Mine.Held
	case KindRally:
		return r.Rally.Held
	}

	return 0
}

// Off is a kind the rules turn off: a grant keeps none of it, and the ones held are kept.
func (r Rules) Off(kind Kind) bool {
	switch kind {
	case KindBomb:
		return r.Bomb.Off
	case KindEncloseClicks:
		return r.Enclose.Off
	case KindSpreadClicks:
		return r.Spread.Off
	case KindRally:
		return r.Rally.Off
	case KindRefill, KindShields, KindVirus, KindMine:
	}

	return false
}

// A tile keeps its shields in a byte.
const mostShieldsOnATile = 255

const MostSpreadRings = 3

const MostMineRings = 3

const (
	widestRally   = 50
	shortestRally = 10 * time.Second
	longestRally  = 30 * time.Minute
	// A click inside a ring costs at least a tenth of one: ten times the clicks at most.
	cheapestRallyClick = 0.1
)

const (
	mostVirusInfections = 1024
	quickestVirus       = time.Second
	slowestVirus        = 10 * time.Minute
)

func (r Rules) Validate() error {
	return errors.Join(
		rules.Positive("refill.held", r.Refill.Held),
		rules.Positive("bomb.held", r.Bomb.Held),
		rules.Positive("virus.held", r.Virus.Held),
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
		r.Spread.validate(),
		rules.Positive("virus.infections", r.Virus.Infections),
		rules.AtMost("virus.infections", r.Virus.Infections, "the most a virus makes", mostVirusInfections),
		rules.AtLeast("virus.every", r.Virus.Every, "a second", quickestVirus),
		rules.AtMost("virus.every", r.Virus.Every, "ten minutes", slowestVirus),
		rules.Positive("mine.held", r.Mine.Held),
		rules.Positive("mine.rings", r.Mine.Rings),
		rules.AtMost("mine.rings", r.Mine.Rings, "the widest blast", MostMineRings),
		rules.Positive("rally.held", r.Rally.Held),
		rules.Positive("rally.rings", r.Rally.Rings), rules.Finite("rally.rings", r.Rally.Rings),
		rules.AtMost("rally.rings", r.Rally.Rings, "the widest ring", widestRally),
		rules.AtLeast("rally.lasts", r.Rally.Lasts, "ten seconds", shortestRally),
		rules.AtMost("rally.lasts", r.Rally.Lasts, "thirty minutes", longestRally),
		rules.AtLeast("rally.cost", r.Rally.Cost, "a tenth of a click", cheapestRallyClick),
		rules.AtMost("rally.cost", r.Rally.Cost, "a whole click", 1),
	)
}

// RefillRules: a hand holds Held refills at most.
type RefillRules struct {
	Held int
}

type SpreadRules struct {
	Clicks    int
	Rings     int
	Unlimited bool
	Off       bool
}

func (r SpreadRules) validate() error {
	if r.Off && r.Unlimited {
		return errors.New("spread.off is on with spread.unlimited: a spread cannot be off and free")
	}
	return nil
}

// BombRules: a hand holds Held bombs at most. Off, a bomb held is not dropped, and a grant keeps none.
type BombRules struct {
	Rings float64
	Off   bool
	Held  int
}

// EncloseRules: Off, a click with enclose on closes nothing and spends nothing, and a grant keeps none. The enclosures
// held are kept.
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

// VirusRules: a hand holds Held viruses at most. A virus makes Infections at most, a tile taken or a shield struck each,
// its first included, one spread every Every.
type VirusRules struct {
	Infections int
	Every      time.Duration
	Held       int
}

// MineRules: a hand holds Held mines at most, and a mine's blast takes Rings rings of tiles around its own.
type MineRules struct {
	Held  int
	Rings int
}

// RallyRules: a hand holds Held rallies at most. A rally's ring is Rings tile spacings wide and lasts Lasts, and a click
// of its flag inside it takes Cost from the bank.
type RallyRules struct {
	Held  int
	Rings float64
	Lasts time.Duration
	Cost  float64
	Off   bool
}

type ShieldRules struct {
	Held    int
	PerTile int
}
