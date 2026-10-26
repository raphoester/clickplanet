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

type Executor interface {
	// Execute answers took false when the set to play is the one in force already.
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
// name. What postgres keeps is the truth, and what is in force follows it.
type UseCase struct {
	sets    Sets
	parts   rules.Parts
	inForce Rules
	bans    Bans

	mu    sync.Mutex
	asked rules.SetName

	taken  rules.Set
	untold bool
}

// Play names the planet rule set in force. The set it names plays at the next Execute.
func (u *UseCase) Play(name rules.SetName) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.asked = name
}

func (u *UseCase) Execute(ctx context.Context) (Taken, bool, error) {
	u.mu.Lock()
	asked := u.asked
	u.mu.Unlock()

	kept, err := u.sets.Set(ctx, asked)
	if errors.Is(err, rules.ErrUnknownSet) && asked != rules.Default {
		kept, err = u.sets.Set(ctx, rules.Default)
	}
	if err != nil {
		return Taken{}, false, fmt.Errorf("could not read the rules to play for %q: %w", asked, err)
	}

	set, err := kept.Restore(u.parts)
	if err != nil {
		return Taken{}, false, fmt.Errorf("the rule set %q kept cannot play: %w", kept.Name, err)
	}

	taken := Taken{Set: set.TakeSnapshot(), Asked: asked}

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
