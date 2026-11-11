package play_maps_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/usecases/play_arenas_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/play_plan_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/usecases/play_resets_usecase"
)

func New(
	planner play_plan_usecase.Planner, resets play_resets_usecase.Executor, arenas play_arenas_usecase.Executor,
) *UseCase {
	return &UseCase{planner: planner, resets: resets, arenas: arenas}
}

// UseCase reads the plan once a tick and hands it to the steps that swap planet's map, a call of seconds: the reset,
// then the arenas. A runner of their own, so a swap holds back no other step; what the others wait on is kept.
type UseCase struct {
	planner play_plan_usecase.Planner
	resets  play_resets_usecase.Executor
	arenas  play_arenas_usecase.Executor
}

var _ play_plan_usecase.Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context) error {
	plan, err := u.planner.Plan(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", play_plan_usecase.ErrUnplanned, err)
	}
	_, _, resetting := u.resets.Execute(ctx, plan)
	_, arenas := u.arenas.Execute(ctx, plan)
	return errors.Join(resetting, arenas)
}
