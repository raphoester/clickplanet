package record_fact_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type Calendar interface {
	DayAt(ctx context.Context, at time.Time) (days.Day, error)
}

type Briefing interface {
	Of(ctx context.Context, day days.Day, account missions.AccountID) (missions.Brief, error)
}

type Sheets interface {
	ChangeSheet(ctx context.Context, day days.Day, account missions.AccountID, change func(missions.Sheet) missions.Sheet) error
}

type Payroll interface {
	Pay(ctx context.Context, day days.Day, account missions.AccountID) error
}

type In struct {
	Account missions.AccountID
	At      time.Time
	Fact    missions.Fact
}

func New(
	calendar Calendar, briefing Briefing, sheets Sheets, transactioner cpatomic.Transactioner, payroll Payroll,
) *UseCase {
	return &UseCase{calendar: calendar, briefing: briefing, sheets: sheets, transactioner: transactioner, payroll: payroll}
}

type UseCase struct {
	calendar      Calendar
	briefing      Briefing
	sheets        Sheets
	transactioner cpatomic.Transactioner
	payroll       Payroll
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	day, err := u.calendar.DayAt(ctx, in.At)
	if errors.Is(err, days.ErrNoDay) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read the day of the fact: %w", err)
	}

	brief, err := u.briefing.Of(ctx, day, in.Account)
	if err != nil {
		return fmt.Errorf("failed to read the account's missions: %w", err)
	}

	return u.transactioner.InTx(ctx, func(ctx context.Context) error {
		finished := false
		if err := u.sheets.ChangeSheet(ctx, day, in.Account, func(sheet missions.Sheet) missions.Sheet {
			next := sheet.WithFact(in.Fact, brief)
			_, dueBefore := sheet.WithPayments(brief)
			_, dueAfter := next.WithPayments(brief)
			finished = len(dueAfter) > len(dueBefore)
			return next
		}); err != nil {
			return fmt.Errorf("failed to count the fact on the day's missions: %w", err)
		}
		if !finished {
			return nil
		}
		return u.payroll.Pay(ctx, day, in.Account) //nolint:wrapcheck // the payroll named it.
	})
}
