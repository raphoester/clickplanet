package rules

import (
	"context"
	"time"
)

// Choice is the set the schedule told planet to play, and when it decided it.
type Choice struct {
	Set       SetName
	DecidedAt time.Time
}

// NoChoice is what planet plays until the schedule names a set.
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
