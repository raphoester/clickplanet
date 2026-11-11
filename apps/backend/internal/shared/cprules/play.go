package cprules

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/protobuf/proto"
)

// Replaying plays the set chosen last again: at boot, and after a save. took is false when the set is in force already.
type Replaying[M proto.Message] interface {
	Execute(ctx context.Context) (taken Snapshot[M], took bool, err error)
}

// Playing also plays a newer choice of the schedule.
type Playing[M proto.Message] interface {
	Replaying[M]
	Play(ctx context.Context, choice Choice) (taken Snapshot[M], took bool, err error)
}

type Enforcer[M proto.Message] interface {
	Enforce(set Set[M])
}

// Hook hears each set taken, and again at each Execute or Play until it answers no error: the antibot tells planet its
// running bans again under the set. Its error fails the call, and names what failed; the set stays taken.
type Hook func(ctx context.Context, taken SetName) error

// NewPlayUseCase takes a nil hook for a module that has nothing to tell: planet and the offers.
func NewPlayUseCase[M proto.Message](sets Store[M], choices Choices, inForce Enforcer[M], hook Hook) *PlayUseCase[M] {
	return &PlayUseCase[M]{sets: sets, choices: choices, inForce: inForce, hook: hook}
}

// PlayUseCase plays the set the schedule chose last, and default until it chooses one.
type PlayUseCase[M proto.Message] struct {
	sets    Store[M]
	choices Choices
	inForce Enforcer[M]
	hook    Hook

	mu     sync.Mutex
	chosen Choice
	taken  Set[M]
	untold bool
}

func (u *PlayUseCase[M]) Execute(ctx context.Context) (Snapshot[M], bool, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	chosen, kept, err := u.choices.Choice(ctx)
	if err != nil {
		return Snapshot[M]{}, false, fmt.Errorf("could not read the rule set chosen last: %w", err)
	}
	if !kept {
		chosen = NoChoice
	}

	set, err := u.sets.Set(ctx, chosen.Set)
	if err != nil {
		return Snapshot[M]{}, false, fmt.Errorf("could not read the rules to play: %w", err)
	}
	u.chosen = chosen

	return u.take(ctx, set)
}

// Play refuses a set the module keeps none of, or one that cannot play: the rules in force stay.
func (u *PlayUseCase[M]) Play(ctx context.Context, choice Choice) (Snapshot[M], bool, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	if !choice.Overrules(u.chosen) {
		return u.tell(ctx, u.taken, false)
	}

	set, err := u.sets.Set(ctx, choice.Set)
	if err != nil {
		return u.taken.TakePlayedSnapshot(), false, fmt.Errorf("could not play the rule set %q: %w", choice.Set, err)
	}
	if err := u.choices.Save(ctx, choice); err != nil {
		return u.taken.TakePlayedSnapshot(), false, fmt.Errorf("could not keep the choice of the rule set %q: %w", choice.Set, err)
	}
	u.chosen = choice

	return u.take(ctx, set)
}

func (u *PlayUseCase[M]) take(ctx context.Context, set Set[M]) (Snapshot[M], bool, error) {
	took := !set.Equal(u.taken)
	if took {
		u.inForce.Enforce(set)
		u.taken, u.untold = set, u.hook != nil
	}
	return u.tell(ctx, set, took)
}

// The hook hears the set taken until it heard it.
func (u *PlayUseCase[M]) tell(ctx context.Context, set Set[M], took bool) (Snapshot[M], bool, error) {
	if u.untold {
		if err := u.hook(ctx, set.name); err != nil {
			return set.TakePlayedSnapshot(), took, err
		}
		u.untold = false
	}
	return set.TakePlayedSnapshot(), took, nil
}
