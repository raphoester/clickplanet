package play_season_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar"
)

type Seasons interface {
	Decision(ctx context.Context, number calendar.Number) (calendar.Decision, bool, error)
	Decisions(ctx context.Context) ([]calendar.Decision, error)
	Save(ctx context.Context, decision calendar.Decision) error
}

// Waker is the runner that closes the rounds and the seasons: a season's end may have moved.
type Waker interface {
	Wake()
}

type Executor interface {
	Execute(ctx context.Context, decision calendar.Decision) (took bool, err error)
}

func New(seasons Seasons, calendar calendar.Calendar, waker Waker) *UseCase {
	return &UseCase{seasons: seasons, calendar: calendar, waker: waker}
}

// UseCase keeps a season's window when it overrules the decision kept, and puts every season kept in force.
type UseCase struct {
	seasons  Seasons
	calendar calendar.Calendar
	waker    Waker
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context, decision calendar.Decision) (bool, error) {
	if err := decision.Window.Check(); err != nil {
		return false, err //nolint:wrapcheck // the window says why.
	}
	kept, found, err := u.seasons.Decision(ctx, decision.Window.Number)
	if err != nil {
		return false, fmt.Errorf("could not read season %d: %w", decision.Window.Number, err)
	}
	if found && !decision.Overrules(kept) {
		return false, nil
	}
	if err := u.seasons.Save(ctx, decision); err != nil {
		return false, fmt.Errorf("could not keep season %d: %w", decision.Window.Number, err)
	}
	decisions, err := u.seasons.Decisions(ctx)
	if err != nil {
		return true, fmt.Errorf("could not read the seasons: %w", err)
	}
	u.calendar.Play(calendar.WindowsOf(decisions)...)
	u.waker.Wake()
	return true, nil
}
