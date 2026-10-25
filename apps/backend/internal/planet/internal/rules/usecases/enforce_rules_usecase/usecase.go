package enforce_rules_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Sets interface {
	Set(ctx context.Context, name rules.SetName) (rules.Set, error)
}

type Layers interface {
	LatestBaseline(ctx context.Context) (rules.Baseline, error)
	Override(ctx context.Context) (rules.Override, error)
}

type Rules interface {
	Enforce(set rules.Set)
}

type Executor interface {
	// Execute answers the set it enforced, and none when the set to play had not changed.
	Execute(ctx context.Context) (rules.Snapshot, error)
}

func New(sets Sets, layers Layers, inForce Rules, clock cptime.Clock) *UseCase {
	return &UseCase{sets: sets, layers: layers, inForce: inForce, clock: clock}
}

// UseCase keeps the game on the set the database says to play now: what is kept is the truth, and what is in force
// follows it, an override's end included.
type UseCase struct {
	sets    Sets
	layers  Layers
	inForce Rules
	clock   cptime.Clock

	enforced rules.Snapshot
}

func (u *UseCase) Execute(ctx context.Context) (rules.Snapshot, error) {
	baseline, err := u.layers.LatestBaseline(ctx)
	if err != nil {
		return rules.Snapshot{}, fmt.Errorf("could not read the baseline: %w", err)
	}
	override, err := u.layers.Override(ctx)
	if err != nil {
		return rules.Snapshot{}, fmt.Errorf("could not read the override: %w", err)
	}

	set, err := u.sets.Set(ctx, rules.LayersOf(baseline, override).SetAt(u.clock.Now()))
	if err != nil {
		return rules.Snapshot{}, fmt.Errorf("could not read the rules to enforce: %w", err)
	}

	kept := set.TakeSnapshot()
	if kept == u.enforced {
		return rules.Snapshot{}, nil
	}

	u.inForce.Enforce(set)
	u.enforced = kept

	return kept, nil
}
