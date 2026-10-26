package enforce_rules_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

type Sets interface {
	Set(ctx context.Context, name rules.SetName) (rules.Set, error)
}

type Rules interface {
	Enforce(set rules.Set)
}

type Executor interface {
	// Execute answers the set it enforced, and none when the set kept had not changed.
	Execute(ctx context.Context) (rules.Snapshot, error)
}

func New(sets Sets, inForce Rules) *UseCase {
	return &UseCase{sets: sets, inForce: inForce}
}

// UseCase keeps the game on the set the database holds: what is kept is the truth, and what is in force follows it.
type UseCase struct {
	sets    Sets
	inForce Rules

	enforced rules.Snapshot
}

func (u *UseCase) Execute(ctx context.Context) (rules.Snapshot, error) {
	set, err := u.sets.Set(ctx, rules.Default)
	if err != nil {
		return rules.Snapshot{}, fmt.Errorf("could not read the rules to enforce: %w", err)
	}

	kept := set.TakeSnapshot()
	if kept.Equal(u.enforced) {
		return rules.Snapshot{}, nil
	}

	u.inForce.Enforce(set)
	u.enforced = kept

	return kept, nil
}
