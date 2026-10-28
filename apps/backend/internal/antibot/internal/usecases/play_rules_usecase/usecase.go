package play_rules_usecase

import (
	"context"
	"fmt"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
)

type Sets interface {
	Set(ctx context.Context, name rules.SetName) (rules.Set, error)
}

type Choices interface {
	Choice(ctx context.Context) (rules.Choice, bool, error)
	Save(ctx context.Context, choice rules.Choice) error
}

type Rules interface {
	Enforce(set rules.Set)
}

type Bans interface {
	Restate(ctx context.Context) error
}

// Execute plays the set chosen last, at boot and after a save; Play plays a newer choice. took is false when the set is
// in force already.
type Executor interface {
	Execute(ctx context.Context) (taken rules.Snapshot, took bool, err error)
	Play(ctx context.Context, choice rules.Choice) (taken rules.Snapshot, took bool, err error)
}

func New(sets Sets, choices Choices, inForce Rules, bans Bans) *UseCase {
	return &UseCase{sets: sets, choices: choices, inForce: inForce, bans: bans}
}

// UseCase plays the set the schedule chose last, and default until it chooses one.
type UseCase struct {
	sets    Sets
	choices Choices
	inForce Rules
	bans    Bans

	mu     sync.Mutex
	chosen rules.Choice
	taken  rules.Set
	untold bool
}

func (u *UseCase) Execute(ctx context.Context) (rules.Snapshot, bool, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	chosen, kept, err := u.choices.Choice(ctx)
	if err != nil {
		return rules.Snapshot{}, false, fmt.Errorf("could not read the rule set chosen last: %w", err)
	}
	if !kept {
		chosen = rules.NoChoice
	}

	set, err := u.sets.Set(ctx, chosen.Set)
	if err != nil {
		return rules.Snapshot{}, false, fmt.Errorf("could not read the rules to play for %q: %w", chosen.Set, err)
	}
	u.chosen = chosen

	return u.take(ctx, set)
}

// Play refuses a set the antibot keeps none of, or one that cannot play: the rules in force stay.
func (u *UseCase) Play(ctx context.Context, choice rules.Choice) (rules.Snapshot, bool, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	if !choice.Overrules(u.chosen) {
		return u.restate(ctx, u.taken, false)
	}

	set, err := u.sets.Set(ctx, choice.Set)
	if err != nil {
		return u.taken.TakeSnapshot(), false, fmt.Errorf("could not play the rule set %q: %w", choice.Set, err)
	}
	if err := u.choices.Save(ctx, choice); err != nil {
		return u.taken.TakeSnapshot(), false, fmt.Errorf("could not keep the choice of the rule set %q: %w", choice.Set, err)
	}
	u.chosen = choice

	return u.take(ctx, set)
}

func (u *UseCase) take(ctx context.Context, set rules.Set) (rules.Snapshot, bool, error) {
	took := !set.Equal(u.taken)
	if took {
		u.inForce.Enforce(set)
		u.taken, u.untold = set, true
	}
	return u.restate(ctx, set, took)
}

// Planet hears each running ban again under the set taken: one that stops enforcing admits them.
func (u *UseCase) restate(ctx context.Context, set rules.Set, took bool) (rules.Snapshot, bool, error) {
	if u.untold {
		if err := u.bans.Restate(ctx); err != nil {
			return set.TakeSnapshot(), took, fmt.Errorf("could not tell planet the running bans under the rule set %q: %w",
				set.TakeSnapshot().Name, err)
		}
		u.untold = false
	}
	return set.TakeSnapshot(), took, nil
}
