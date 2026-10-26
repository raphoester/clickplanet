package play_rules_usecase

import (
	"context"
	"fmt"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

type Sets interface {
	Set(ctx context.Context, name rules.SetName) (rules.Snapshot, error)
}

type Rules interface {
	Enforce(set rules.Set)
}

// Execute plays the set the game plays; took is false when it is in force already.
type Executor interface {
	Execute(ctx context.Context) (taken rules.Snapshot, took bool, err error)
}

func New(sets Sets, parts rules.Parts, inForce Rules) *UseCase {
	return &UseCase{sets: sets, parts: parts, inForce: inForce}
}

// UseCase plays default: the game plays no other set until a schedule names one.
type UseCase struct {
	sets    Sets
	parts   rules.Parts
	inForce Rules

	mu    sync.Mutex
	taken rules.Set
}

// Plays is true for the set in force.
func (u *UseCase) Plays(name rules.SetName) bool {
	u.mu.Lock()
	defer u.mu.Unlock()

	return u.taken.Is(name)
}

func (u *UseCase) Execute(ctx context.Context) (rules.Snapshot, bool, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	kept, err := u.sets.Set(ctx, rules.Default)
	if err != nil {
		return rules.Snapshot{}, false, fmt.Errorf("could not read the rules to play: %w", err)
	}

	set, err := kept.Restore(u.parts)
	if err != nil {
		return rules.Snapshot{}, false, fmt.Errorf("the rule set %q kept cannot play: %w", kept.Name, err)
	}

	took := !set.Equal(u.taken)
	if took {
		u.inForce.Enforce(set)
		u.taken = set
	}

	return set.TakeSnapshot(), took, nil
}
