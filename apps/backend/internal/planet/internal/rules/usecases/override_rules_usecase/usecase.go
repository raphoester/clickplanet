package override_rules_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Sets interface {
	Set(ctx context.Context, name rules.SetName) (rules.Set, error)
}

type Overrides interface {
	SaveOverride(ctx context.Context, override rules.Override) error
}

type In struct {
	Set   rules.SetName
	Until time.Time
}

func New(sets Sets, overrides Overrides, clock cptime.Clock) *UseCase {
	return &UseCase{sets: sets, overrides: overrides, clock: clock}
}

// UseCase puts a set over the baseline from now, in place of any override running.
type UseCase struct {
	sets      Sets
	overrides Overrides
	clock     cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	set, err := u.sets.Set(ctx, in.Set)
	if err != nil {
		return fmt.Errorf("cannot override the rules with %q: %w", in.Set, err)
	}

	override, err := rules.NewOverride(set, u.clock.Now(), in.Until)
	if err != nil {
		return fmt.Errorf("cannot override the rules with %q: %w", in.Set, err)
	}

	if err := u.overrides.SaveOverride(ctx, override); err != nil {
		return fmt.Errorf("could not keep the override with %q: %w", in.Set, err)
	}

	return nil
}
