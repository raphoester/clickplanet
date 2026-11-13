package bonuses

import (
	"errors"
	"fmt"
	"time"
)

// Rules is what the offers play (bonuses.v1.Rules): when a box comes, and what a box or a right answer may be and how
// much it grants.
// How much of each charge a hand holds, and what it does when it is spent, is planet's: an offer pays what it drew, and
// the wallet keeps what fits.
type Rules struct {
	Boxes Pace

	Kinds map[Kind]float64

	OfferTTL     time.Duration
	MissRetry    time.Duration
	ActiveWithin time.Duration

	Spread  SpreadRules
	Enclose EncloseRules
	Shield  ShieldRules

	Welcome WelcomeRules

	SignInRefill bool

	// Frozen offers nothing, and refuses every claim.
	Frozen bool
}

func (r Rules) Validate() error {
	return errors.Join(
		positive("boxes.minInterval", r.Boxes.MinInterval),
		atLeast("boxes.maxInterval", r.Boxes.MaxInterval, "boxes.minInterval", r.Boxes.MinInterval),
		positive("boxes.maxChargesPerHour", r.Boxes.MaxChargesPerHour),
		weightsProblem(r),
		positive("offerTtl", r.OfferTTL),
		positive("missRetry", r.MissRetry),
		positive("activeWithin", r.ActiveWithin),
		positive("spread.maxPerBox", r.Spread.MaxPerBox),
		positive("enclose.maxPerBox", r.Enclose.MaxPerBox),
		positive("shield.minPerBox", r.Shield.MinPerBox),
		atLeast("shield.maxPerBox", r.Shield.MaxPerBox, "shield.minPerBox", r.Shield.MinPerBox),
		notNegative("welcome.within", r.Welcome.Within),
		notNegative("welcome.boxAfter", r.Welcome.BoxAfter),
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

type SpreadRules struct {
	MaxPerBox int
}

// Offered is true for a kind a box or a right answer may be: weighed above 0. A bomb while planet's bombs are off is a kind the
// wallet says no hand has room for.
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

type WelcomeRules struct {
	Within time.Duration

	BoxAfter time.Duration
}

// ErrFrozen refuses a claim while the offers play a frozen set or planet's map takes no play.
var ErrFrozen = errors.New("no box can be taken now: the game is frozen")
