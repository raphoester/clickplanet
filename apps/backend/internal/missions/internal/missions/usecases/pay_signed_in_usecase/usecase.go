package pay_signed_in_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Payroll interface {
	Pay(ctx context.Context, day missions.Day, account missions.AccountID) error
}

func New(calendar missions.Calendar, clock cptime.Clock, transactioner cpatomic.Transactioner, payroll Payroll) *UseCase {
	return &UseCase{calendar: calendar, clock: clock, transactioner: transactioner, payroll: payroll}
}

type UseCase struct {
	calendar      missions.Calendar
	clock         cptime.Clock
	transactioner cpatomic.Transactioner
	payroll       Payroll
}

func (u *UseCase) Execute(ctx context.Context, account missions.AccountID) error {
	day := u.calendar.DayOf(u.clock.Now())

	return u.transactioner.InTx(ctx, func(ctx context.Context) error { //nolint:wrapcheck // the payroll named it.
		return u.payroll.Pay(ctx, day, account)
	})
}
