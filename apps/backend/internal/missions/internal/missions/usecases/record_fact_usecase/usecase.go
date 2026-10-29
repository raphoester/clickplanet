package record_fact_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type Briefing interface {
	Of(ctx context.Context, day missions.Day, account missions.AccountID) (missions.Brief, error)
}

type Sheets interface {
	ChangeSheet(ctx context.Context, day missions.Day, account missions.AccountID, change func(missions.Sheet) missions.Sheet) error
}

type Payroll interface {
	Pay(ctx context.Context, day missions.Day, account missions.AccountID) error
}

type In struct {
	Account missions.AccountID
	At      time.Time
	Fact    missions.Fact
}

func New(
	calendar missions.Calendar, briefing Briefing, sheets Sheets, transactioner cpatomic.Transactioner, payroll Payroll,
) *UseCase {
	return &UseCase{calendar: calendar, briefing: briefing, sheets: sheets, transactioner: transactioner, payroll: payroll}
}

type UseCase struct {
	calendar      missions.Calendar
	briefing      Briefing
	sheets        Sheets
	transactioner cpatomic.Transactioner
	payroll       Payroll
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	day := u.calendar.DayOf(in.At)

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
