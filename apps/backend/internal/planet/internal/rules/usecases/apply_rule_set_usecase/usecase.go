package apply_rule_set_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Sets interface {
	Set(ctx context.Context, name rules.SetName) (rules.Set, error)
}

type Baselines interface {
	Baseline(ctx context.Context, request rules.RequestID) (rules.Baseline, error)
	SaveBaseline(ctx context.Context, baseline rules.Baseline) error
}

type In struct {
	Request rules.RequestID
	Set     rules.SetName
}

func New(sets Sets, baselines Baselines, clock cptime.Clock) *UseCase {
	return &UseCase{sets: sets, baselines: baselines, clock: clock}
}

// UseCase keeps a set as the baseline; the game follows it within a second, while no override runs.
type UseCase struct {
	sets      Sets
	baselines Baselines
	clock     cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	set, err := u.sets.Set(ctx, in.Set)
	if err != nil {
		return fmt.Errorf("cannot apply request %q: %w", in.Request, err)
	}

	// TODO(broker): this dedupe stands in for a message broker's: a caller sends its request again until it is answered.
	kept, err := u.baselines.Baseline(ctx, in.Request)
	if err != nil {
		return fmt.Errorf("could not tell whether request %q was applied: %w", in.Request, err)
	}

	baseline, err := rules.NewBaseline(in.Request, set, u.clock.Now(), kept)
	if errors.Is(err, rules.ErrAppliedAlready) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("cannot apply request %q: %w", in.Request, err)
	}

	if err := u.baselines.SaveBaseline(ctx, baseline); err != nil {
		return fmt.Errorf("could not keep request %q: %w", in.Request, err)
	}

	return nil
}
