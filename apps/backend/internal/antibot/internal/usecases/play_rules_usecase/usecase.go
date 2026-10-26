package play_rules_usecase

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
)

type Sets interface {
	Set(ctx context.Context, name rules.SetName) (rules.Snapshot, error)
}

type Rules interface {
	Enforce(set rules.Set)
}

type Bans interface {
	Restate(ctx context.Context) error
}

// Execute plays the set for the name asked last; took is false when it is in force already.
type Executor interface {
	Execute(ctx context.Context) (taken Taken, took bool, err error)
}

type Taken struct {
	Set rules.Snapshot
	// The planet rule set the antibot was asked to play.
	Asked rules.SetName
}

func New(sets Sets, parts rules.Parts, inForce Rules, bans Bans) *UseCase {
	return &UseCase{sets: sets, parts: parts, inForce: inForce, bans: bans, asked: rules.Default}
}

// UseCase plays the set named like the planet rule set in force, or default when the antibot keeps no set of that
// name.
type UseCase struct {
	sets    Sets
	parts   rules.Parts
	inForce Rules
	bans    Bans

	mu     sync.Mutex
	asked  rules.SetName
	taken  rules.Set
	untold bool
}

func (u *UseCase) Play(ctx context.Context, name rules.SetName) (Taken, bool, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.asked = name
	return u.playLocked(ctx)
}

// Plays is true for the set in force, and for the set named like planet's even before the antibot keeps one.
func (u *UseCase) Plays(name rules.SetName) bool {
	u.mu.Lock()
	defer u.mu.Unlock()

	return name == u.asked || u.taken.Is(name)
}

func (u *UseCase) Execute(ctx context.Context) (Taken, bool, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	return u.playLocked(ctx)
}

func (u *UseCase) playLocked(ctx context.Context) (Taken, bool, error) {
	kept, err := u.sets.Set(ctx, u.asked)
	if errors.Is(err, rules.ErrUnknownSet) && u.asked != rules.Default {
		kept, err = u.sets.Set(ctx, rules.Default)
	}
	if err != nil {
		return Taken{}, false, fmt.Errorf("could not read the rules to play for %q: %w", u.asked, err)
	}

	set, err := kept.Restore(u.parts)
	if err != nil {
		return Taken{}, false, fmt.Errorf("the rule set %q kept cannot play: %w", kept.Name, err)
	}

	taken := Taken{Set: set.TakeSnapshot(), Asked: u.asked}

	took := !set.Equal(u.taken)
	if took {
		u.inForce.Enforce(set)
		u.taken, u.untold = set, true
	}

	// Planet hears each running ban again under the set taken: one that stops enforcing admits them.
	if u.untold {
		if err := u.bans.Restate(ctx); err != nil {
			return taken, took, fmt.Errorf("could not tell planet the running bans under the rule set %q: %w",
				taken.Set.Name, err)
		}
		u.untold = false
	}

	return taken, took, nil
}
