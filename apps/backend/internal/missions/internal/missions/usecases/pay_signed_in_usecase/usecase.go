package pay_signed_in_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Calendar interface {
	DayAt(ctx context.Context, at time.Time) (days.Day, error)
}

type Payroll interface {
	Pay(ctx context.Context, day days.Day, account missions.AccountID) error
}

func New(calendar Calendar, clock cptime.Clock, transactioner cpatomic.Transactioner, payroll Payroll) *UseCase {
	return &UseCase{calendar: calendar, clock: clock, transactioner: transactioner, payroll: payroll}
}

type UseCase struct {
	calendar      Calendar
	clock         cptime.Clock
	transactioner cpatomic.Transactioner
	payroll       Payroll
}

func (u *UseCase) Execute(ctx context.Context, account missions.AccountID) error {
	day, err := u.calendar.DayAt(ctx, u.clock.Now())
	if errors.Is(err, days.ErrNoDay) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read the day: %w", err)
	}

	return u.transactioner.InTx(ctx, func(ctx context.Context) error { //nolint:wrapcheck // the payroll named it.
		return u.payroll.Pay(ctx, day, account)
	})
}
