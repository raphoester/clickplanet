package boxes

import (
	"errors"
	"time"
)

// Rules is what the boxes play (boxes.v1.Rules): when a box comes, how long it flies, and a newcomer's first box. What
// a box holds is the bonuses module's.
type Rules struct {
	Pace Pace

	OfferTTL     time.Duration
	MissRetry    time.Duration
	ActiveWithin time.Duration

	Welcome WelcomeRules

	// Frozen offers nothing, and refuses every claim.
	Frozen bool
}

func (r Rules) Validate() error {
	return errors.Join(
		positive("minInterval", r.Pace.MinInterval),
		atLeast("maxInterval", r.Pace.MaxInterval, "minInterval", r.Pace.MinInterval),
		positive("maxBoxesPerHour", r.Pace.MaxBoxesPerHour),
		positive("offerTtl", r.OfferTTL),
		positive("missRetry", r.MissRetry),
		positive("activeWithin", r.ActiveWithin),
		notNegative("welcome.within", r.Welcome.Within),
		notNegative("welcome.boxAfter", r.Welcome.BoxAfter),
	)
}

type WelcomeRules struct {
	Within time.Duration

	BoxAfter time.Duration
}

// ErrFrozen refuses a claim while the boxes play a frozen set or planet's map takes no play.
var ErrFrozen = errors.New("no box can be taken now: the game is frozen")
