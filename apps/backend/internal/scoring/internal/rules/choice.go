package rules

import (
	"context"
	"time"
)

// Window is the time of the schedule entry that plays a set. It never changes once the entry runs.
type Window struct {
	StartsAt time.Time
	EndsAt   time.Time
}

// None is true when no entry runs and default plays.
func (w Window) None() bool { return w.EndsAt.IsZero() }

// Choice is the set the schedule told scoring to play, its entry's window, and when the schedule decided it.
type Choice struct {
	Set       SetName
	Window    Window
	DecidedAt time.Time
}

// NoChoice is what scoring plays until the schedule names a set.
var NoChoice = Choice{Set: Default}

// Overrules is true for a choice decided after the one kept: one sent again, or older, changes nothing.
func (c Choice) Overrules(kept Choice) bool {
	return c.DecidedAt.After(kept.DecidedAt)
}

type Choices interface {
	// Choice is the choice kept last; false when none is.
	Choice(ctx context.Context) (Choice, bool, error)
	Save(ctx context.Context, choice Choice) error
}
