package record_fact_usecase

import (
	"context"
	"fmt"
	"slices"
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

type Streams interface {
	Publish(account missions.AccountID)
}

type In struct {
	Account missions.AccountID
	At      time.Time
	Fact    missions.Fact
}

func New(
	calendar Calendar, briefing Briefing, sheets Sheets, transactioner cpatomic.Transactioner, payroll Payroll,
	streams Streams,
) *UseCase {
	return &UseCase{
		calendar: calendar, briefing: briefing, sheets: sheets, transactioner: transactioner, payroll: payroll,
		streams: streams,
	}
}

type UseCase struct {
	calendar      Calendar
	briefing      Briefing
	sheets        Sheets
	transactioner cpatomic.Transactioner
	payroll       Payroll
	streams       Streams
}

// Execute counts each fact on the missions of its account's day: one transaction per day and account, however many
// facts, so an enclose of a continent is one write. Each account's streams wake once, after.
func (u *UseCase) Execute(ctx context.Context, facts ...In) error {
	groups, err := days.GroupsOf(ctx, u.calendar, facts, func(in In) (time.Time, missions.AccountID) { return in.At, in.Account })
	if err != nil {
		return fmt.Errorf("failed to read the days of the facts: %w", err)
	}

	woken := make([]missions.AccountID, 0, len(groups))
	for _, group := range groups {
		if err := u.record(ctx, group); err != nil {
			return err
		}
		if !slices.Contains(woken, group.Key) {
			woken = append(woken, group.Key)
		}
	}
	for _, account := range woken {
		u.streams.Publish(account)
	}
	return nil
}

func (u *UseCase) record(ctx context.Context, group days.Group[missions.AccountID, In]) error {
	brief, err := u.briefing.Of(ctx, group.Day, group.Key)
	if err != nil {
		return fmt.Errorf("failed to read the account's missions: %w", err)
	}

	return u.transactioner.InTx(ctx, func(ctx context.Context) error {
		finished := false
		if err := u.sheets.ChangeSheet(ctx, group.Day, group.Key, func(sheet missions.Sheet) missions.Sheet {
			next := sheet
			for _, in := range group.Facts {
				next = next.WithFact(in.Fact, brief)
			}
			_, dueBefore := sheet.WithPayments(brief)
			_, dueAfter := next.WithPayments(brief)
			finished = len(dueAfter) > len(dueBefore)
			return next
		}); err != nil {
			return fmt.Errorf("failed to count the facts on the day's missions: %w", err)
		}
		if !finished {
			return nil
		}
		return u.payroll.Pay(ctx, group.Day, group.Key) //nolint:wrapcheck // the payroll named it.
	})
}
