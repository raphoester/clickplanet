package rules

import (
	"context"
	"errors"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds"
)

// Window is the schedule entry that plays a set, and its time. It never changes once the entry runs.
type Window struct {
	Entry    rounds.EntryID
	StartsAt time.Time
	EndsAt   time.Time
}

// None is true when no entry runs and default plays.
func (w Window) None() bool { return w.EndsAt.IsZero() }

// ErrNoEntry refuses a window told with no entry's id: its round of its own could not repeat it.
var ErrNoEntry = errors.New("a window comes with the id of its entry")

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
