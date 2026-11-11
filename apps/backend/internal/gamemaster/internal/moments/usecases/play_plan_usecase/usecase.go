package play_plan_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/usecases/judge_battles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/play_scoring_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/tell_starts_usecase"
)

type Planner interface {
	Plan(ctx context.Context) (moments.Plan, error)
}

var ErrUnplanned = errors.New("could not plan the books")

type Executor interface {
	Execute(ctx context.Context) error
}

func New(
	planner Planner, judge judge_battles_usecase.Executor, scoring play_scoring_usecase.Executor,
	starts tell_starts_usecase.Executor,
) *UseCase {
	return &UseCase{planner: planner, judge: judge, scoring: scoring, starts: starts}
}

// UseCase reads the plan once a tick and hands it to each step in turn: the battle's judge, scoring, then what starts. A
// step that fails stops none after it, and the steps after the judge get the battle it judged.
type UseCase struct {
	planner Planner
	judge   judge_battles_usecase.Executor
	scoring play_scoring_usecase.Executor
	starts  tell_starts_usecase.Executor
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context) error {
	plan, err := u.planner.Plan(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnplanned, err)
	}
	judged, judging := u.judge.Execute(ctx, plan)
	plan = plan.Judged(judged...)
	_, scoring := u.scoring.Execute(ctx, plan)
	_, starting := u.starts.Execute(ctx, plan)
	return errors.Join(judging, scoring, starting)
}
