package play_rules_usecase

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rules"
)

type Sets interface {
	Set(ctx context.Context, name rules.SetName) (rules.Set, error)
}

type Choices interface {
	Choice(ctx context.Context) (rules.Choice, bool, error)
	Save(ctx context.Context, choice rules.Choice) error
}

// Opener opens the round of its own a set asks of the window of the entry that plays it.
type Opener interface {
	Execute(ctx context.Context, times uint32, startsAt, endsAt time.Time) error
}

// Played is the set scoring counts with, and the window of the entry that plays it.
type Played struct {
	Set    rules.Snapshot
	Window rules.Window
}

// Execute plays the set chosen last, at boot; Play plays a newer choice. took is false when the choice is not newer.
type Executor interface {
	Execute(ctx context.Context) (played Played, took bool, err error)
	Play(ctx context.Context, choice rules.Choice) (played Played, took bool, err error)
}

func New(sets Sets, choices Choices, opener Opener) *UseCase {
	return &UseCase{sets: sets, choices: choices, opener: opener}
}

// UseCase plays the set the schedule chose last, and default until it chooses one. A set with a round of its own opens
// it before the choice is kept, so a choice that failed to open it is played again when its command comes again.
type UseCase struct {
	sets    Sets
	choices Choices
	opener  Opener

	mu     sync.Mutex
	chosen rules.Choice
	taken  rules.Set
}

// Plays is true for the set in force.
func (u *UseCase) Plays(name rules.SetName) bool {
	u.mu.Lock()
	defer u.mu.Unlock()

	return u.taken.Is(name)
}

// Name is the name of the set in force.
func (u *UseCase) Name() rules.SetName {
	u.mu.Lock()
	defer u.mu.Unlock()

	return u.chosen.Set
}

func (u *UseCase) Execute(ctx context.Context) (Played, bool, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	chosen, kept, err := u.choices.Choice(ctx)
	if err != nil {
		return Played{}, false, fmt.Errorf("could not read the rule set chosen last: %w", err)
	}
	if !kept {
		chosen = rules.NoChoice
	}

	set, err := u.sets.Set(ctx, chosen.Set)
	if err != nil {
		return Played{}, false, fmt.Errorf("could not read the rules to play: %w", err)
	}
	if err := u.open(ctx, set, chosen.Window); err != nil {
		return Played{}, false, err
	}
	u.chosen, u.taken = chosen, set

	return u.played(), true, nil
}

// Play refuses a set scoring keeps none of, or one that cannot play: the rules in force stay.
func (u *UseCase) Play(ctx context.Context, choice rules.Choice) (Played, bool, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	if !choice.Overrules(u.chosen) {
		return u.played(), false, nil
	}

	set, err := u.sets.Set(ctx, choice.Set)
	if err != nil {
		return u.played(), false, fmt.Errorf("could not play the rule set %q: %w", choice.Set, err)
	}
	if err := u.open(ctx, set, choice.Window); err != nil {
		return u.played(), false, err
	}
	if err := u.choices.Save(ctx, choice); err != nil {
		return u.played(), false, fmt.Errorf("could not keep the choice of the rule set %q: %w", choice.Set, err)
	}
	u.chosen, u.taken = choice, set

	return u.played(), true, nil
}

func (u *UseCase) open(ctx context.Context, set rules.Set, window rules.Window) error {
	own := set.OwnRound()
	if !own.Enabled || window.None() {
		return nil
	}
	if err := u.opener.Execute(ctx, own.Times, window.StartsAt, window.EndsAt); err != nil {
		return fmt.Errorf("could not open the round of its own of the rule set %q: %w", set.TakeSnapshot().Name, err)
	}
	return nil
}

func (u *UseCase) played() Played {
	return Played{Set: u.taken.TakeSnapshot(), Window: u.chosen.Window}
}
