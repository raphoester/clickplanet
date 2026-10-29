package tell_day_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Planner interface {
	Plan(ctx context.Context) (moments.Plan, error)
}

type Told interface {
	KeepTold(ctx context.Context, day moments.ToldDay) (bool, error)
}

// Execute answers the day it told this time, if any.
type Executor interface {
	Execute(ctx context.Context) ([]moments.DayStarted, error)
}

func New(
	planner Planner, told Told, events moments.Events, transactioner cpatomic.Transactioner, clock cptime.Clock,
) *UseCase {
	return &UseCase{planner: planner, told: told, events: events, transactioner: transactioner, clock: clock}
}

// UseCase tells the day that runs, once, and again when its end moved. Level-triggered: a boot tells the day that runs
// if it was never told.
type UseCase struct {
	planner       Planner
	told          Told
	events        moments.Events
	transactioner cpatomic.Transactioner
	clock         cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context) ([]moments.DayStarted, error) {
	plan, err := u.planner.Plan(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not plan the books: %w", err)
	}
	day, ok := plan.DayAt(u.clock.Now())
	if !ok {
		return nil, nil
	}

	started := moments.DayStarted{Day: day}
	var told []moments.DayStarted
	err = u.transactioner.InTx(ctx, func(ctx context.Context) error {
		kept, err := u.told.KeepTold(ctx, moments.ToldDay{Round: moments.DayRoundOf(day), EndsAt: day.EndsAt})
		if err != nil || !kept {
			return err //nolint:wrapcheck // the store names the day.
		}
		told = append(told, started)
		return u.events.Record(ctx, started) //nolint:wrapcheck // the adapter names the event.
	})
	if err != nil {
		return nil, err //nolint:wrapcheck // the transaction's own error.
	}
	return told, nil
}
