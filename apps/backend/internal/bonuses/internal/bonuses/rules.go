package bonuses

import (
	"errors"
	"fmt"
)

// Rules is what the bonuses module plays (bonuses.v1.Rules): what a box caught or a right answer may be and how much it
// grants, its gates, and the sign-in refill. How much of each charge a hand holds, and what it does when it is spent, is
// planet's: a win pays what it drew, and the wallet keeps what fits.
type Rules struct {
	Kinds map[Kind]float64
	Gates GateRules

	Spread  SpreadRules
	Enclose EncloseRules
	Shield  ShieldRules
	Mine    MineRules

	SignInRefill bool
}

func (r Rules) Validate() error {
	return errors.Join(
		weightsProblem(r),
		gatesProblem(r),
		positive("spread.maxPerBox", r.Spread.MaxPerBox),
		positive("enclose.maxPerBox", r.Enclose.MaxPerBox),
		positive("shield.minPerBox", r.Shield.MinPerBox),
		atLeast("shield.maxPerBox", r.Shield.MaxPerBox, "shield.minPerBox", r.Shield.MinPerBox),
		positive("mine.maxPerBox", r.Mine.MaxPerBox),
	)
}

// Each kind by its field in bonuses.v1.KindWeights.
var weighed = []struct {
	kind  Kind
	field string
}{
	{KindRefill, "refill"},
	{KindSpreadClicks, "spreadClicks"},
	{KindEncloseClicks, "encloseClicks"},
	{KindBomb, "bomb"},
	{KindShields, "shields"},
	{KindVirus, "virus"},
	{KindMine, "mine"},
}

func weightsProblem(r Rules) error {
	var problems []error
	offered := false
	for _, each := range weighed {
		weight := r.Kinds[each.kind]
		problems = append(problems, notNegative("kinds."+each.field, weight), finite("kinds."+each.field, weight))
		offered = offered || r.Offered(each.kind)
	}
	if !offered {
		problems = append(problems, fmt.Errorf("kinds weigh every kind a box may be 0, so no box could be anything"))
	}

	return errors.Join(problems...)
}

// GateRules is whether a win is drawn only from the kinds open to everyone and the ones the account unlocked, or from
// every kind. Gates decide what is drawn, never what a hand may spend.
type GateRules struct {
	On   bool
	Open map[Kind]bool
}

// MayDraw is true for a kind a win of this account may be: any while the gates are off, else an open one or one
// progression unlocked for it. A guest has none unlocked.
func (r Rules) MayDraw(kind Kind, unlocked Unlocked) bool {
	return !r.Gates.On || r.Gates.Open[kind] || unlocked.Holds(kind)
}

// A guest draws only the open kinds, so gates whose open kinds all weigh 0 pay a guest's box nothing.
func gatesProblem(r Rules) error {
	if !r.Gates.On {
		return nil
	}
	for _, each := range weighed {
		if r.Offered(each.kind) && r.Gates.Open[each.kind] {
			return nil
		}
	}

	return fmt.Errorf("gates.on with no kind gates.open opens weighed above 0, so a guest's box could be nothing")
}

type SpreadRules struct {
	MaxPerBox int
}

// Offered is true for a kind a box or a right answer may be: weighed above 0. A bomb while planet's bombs are off is a
// kind the wallet says no hand has room for.
func (r Rules) Offered(kind Kind) bool {
	return r.Kinds[kind] > 0
}

type EncloseRules struct {
	MaxPerBox int
}

type ShieldRules struct {
	MinPerBox int
	MaxPerBox int
}

type MineRules struct {
	MaxPerBox int
}
