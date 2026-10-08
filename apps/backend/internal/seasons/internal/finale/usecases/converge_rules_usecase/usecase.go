package converge_rules_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/finale"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Planet interface {
	Set(ctx context.Context, switches finale.Switches) error
}

type Executor interface {
	Execute(ctx context.Context) error
}

func New(seasons calendar.Calendar, rules finale.Rules, clock cptime.Clock, planet Planet) *UseCase {
	return &UseCase{seasons: seasons, rules: rules, clock: clock, planet: planet}
}

type UseCase struct {
	seasons calendar.Calendar
	rules   finale.Rules
	clock   cptime.Clock
	planet  Planet

	confirmed *finale.Switches
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context) error {
	wanted := finale.PhaseAt(u.seasons, u.clock.Now()).Switches(u.rules)
	if u.confirmed != nil && u.confirmed.Equal(wanted) {
		return nil
	}

	if err := u.planet.Set(ctx, wanted); err != nil {
		return fmt.Errorf("failed to set the rules the calendar holds now: %w", err)
	}

	u.confirmed = &wanted
	return nil
}
