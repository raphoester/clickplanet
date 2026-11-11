package judge_battles_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Judged interface {
	KeepJudged(ctx context.Context, judgement moments.Judgement) (bool, error)
}

// Planet says whether a rule set makes an arena battle.
type Planet interface {
	Arena(ctx context.Context, set schedule.SetName) (bool, error)
}

// Execute answers the battle it judged this time, if any.
type Executor interface {
	Execute(ctx context.Context, plan moments.Plan) ([]moments.Judgement, error)
}

func New(judged Judged, planet Planet, clock cptime.Clock) *UseCase {
	return &UseCase{judged: judged, planet: planet, clock: clock}
}

// UseCase judges the battle that runs, once, from its set: a set edited later changes the judgement no more.
type UseCase struct {
	judged Judged
	planet Planet
	clock  cptime.Clock
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context, plan moments.Plan) ([]moments.Judgement, error) {
	battle, ok := plan.Unjudged(u.clock.Now())
	if !ok {
		return nil, nil
	}
	inArena, err := u.planet.Arena(ctx, battle.Set)
	if err != nil {
		return nil, err //nolint:wrapcheck // the adapter names the set.
	}
	judgement := plan.Judge(battle, inArena)
	if _, err := u.judged.KeepJudged(ctx, judgement); err != nil {
		return nil, fmt.Errorf("failed to keep %s judged: %w", battle, err)
	}
	return []moments.Judgement{judgement}, nil
}
