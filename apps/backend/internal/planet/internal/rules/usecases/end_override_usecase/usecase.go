package end_override_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Overrides interface {
	Override(ctx context.Context) (rules.Override, error)
	SaveOverride(ctx context.Context, override rules.Override) error
}

func New(overrides Overrides, clock cptime.Clock) *UseCase {
	return &UseCase{overrides: overrides, clock: clock}
}

// UseCase ends the override running now, so the baseline plays again; with none running, it changes nothing.
type UseCase struct {
	overrides Overrides
	clock     cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context) error {
	override, err := u.overrides.Override(ctx)
	if err != nil {
		return fmt.Errorf("could not read the override to end: %w", err)
	}

	ended := override.EndedAt(u.clock.Now())
	if ended == override {
		return nil
	}

	if err := u.overrides.SaveOverride(ctx, ended); err != nil {
		return fmt.Errorf("could not end the override: %w", err)
	}

	return nil
}
