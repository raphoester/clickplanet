package record_fact_usecase

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/donemessage"
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

type Done interface {
	DoneBy(ctx context.Context, account missions.AccountID) (missions.Done, error)
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
	calendar Calendar, briefing Briefing, sheets Sheets, done Done, transactioner cpatomic.Transactioner,
	outbox cpatomic.Outbox, payroll Payroll, streams Streams,
) *UseCase {
	return &UseCase{
		calendar: calendar, briefing: briefing, sheets: sheets, done: done, transactioner: transactioner, outbox: outbox,
		payroll: payroll, streams: streams,
	}
}

type UseCase struct {
	calendar      Calendar
	briefing      Briefing
	sheets        Sheets
	done          Done
	transactioner cpatomic.Transactioner
	outbox        cpatomic.Outbox
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
		if err := u.tellDone(ctx, group.Key); err != nil {
			return err
		}
		return u.payroll.Pay(ctx, group.Day, group.Key) //nolint:wrapcheck // the payroll named it.
	})
}

// What the account has done in all, these included, told whether or not it is signed in: paying is the payroll's
// business, and what a mission done is worth the listener's.
func (u *UseCase) tellDone(ctx context.Context, account missions.AccountID) error {
	done, err := u.done.DoneBy(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to count the account's missions done: %w", err)
	}
	if err := u.outbox.Record(ctx, donemessage.Of(account, done)); err != nil {
		return fmt.Errorf("failed to record the missions done: %w", err)
	}
	return nil
}
