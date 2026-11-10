package play_plan_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/usecases/play_arenas_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/play_scoring_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/tell_starts_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/usecases/play_resets_usecase"
)

type Planner interface {
	Plan(ctx context.Context) (moments.Plan, error)
}

var ErrUnplanned = errors.New("could not plan the books")

type Executor interface {
	Execute(ctx context.Context) error
}

func New(
	planner Planner, resets play_resets_usecase.Executor, arenas play_arenas_usecase.Executor,
	scoring play_scoring_usecase.Executor, starts tell_starts_usecase.Executor,
) *UseCase {
	return &UseCase{planner: planner, resets: resets, arenas: arenas, scoring: scoring, starts: starts}
}

// UseCase reads the plan once a tick and hands it to each step in turn: the reset, the arena battles, scoring, then what
// starts. A step that fails stops none after it, and the steps after the arena battles' get the battle it judged.
type UseCase struct {
	planner Planner
	resets  play_resets_usecase.Executor
	arenas  play_arenas_usecase.Executor
	scoring play_scoring_usecase.Executor
	starts  tell_starts_usecase.Executor
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context) error {
	plan, err := u.planner.Plan(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnplanned, err)
	}
	_, _, resetting := u.resets.Execute(ctx, plan)
	tick, arenas := u.arenas.Execute(ctx, plan)
	plan = plan.Judged(tick.Judged...)
	_, scoring := u.scoring.Execute(ctx, plan)
	_, starting := u.starts.Execute(ctx, plan)
	return errors.Join(resetting, arenas, scoring, starting)
}
