package record_strike_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions/flagmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions/scoringgrant"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type Calendar interface {
	DayAt(ctx context.Context, at time.Time) (days.Day, error)
}

type Briefing interface {
	Of(ctx context.Context, day days.Day, flag string) (flagmissions.Day, error)
}

type FlagDays interface {
	ChangeDay(ctx context.Context, day days.Day, flag string, change func(flagmissions.Day) flagmissions.Day) error
}

type Streams interface {
	Publish(account missions.AccountID)
}

func New(
	calendar Calendar, briefing Briefing, flagDays FlagDays, transactioner cpatomic.Transactioner, outbox cpatomic.Outbox,
	streams Streams,
) *UseCase {
	return &UseCase{
		calendar: calendar, briefing: briefing, flagDays: flagDays, transactioner: transactioner, outbox: outbox,
		streams: streams,
	}
}

type UseCase struct {
	calendar      Calendar
	briefing      Briefing
	flagDays      FlagDays
	transactioner cpatomic.Transactioner
	outbox        cpatomic.Outbox
	streams       Streams
}

// Execute counts each shield broken on its flag's day: one transaction per day and flag, however many shields, which
// records the points and the word of a mission they finished. Each account that broke one wakes once, after.
func (u *UseCase) Execute(ctx context.Context, strikes ...flagmissions.Strike) error {
	groups, err := days.GroupsOf(ctx, u.calendar, strikes, func(strike flagmissions.Strike) (time.Time, string) {
		return strike.At, strike.Flag
	})
	if err != nil {
		return fmt.Errorf("failed to read the days of the shields broken: %w", err)
	}

	var woken []missions.AccountID
	seen := map[missions.AccountID]bool{}
	for _, group := range groups {
		if err := u.record(ctx, group); err != nil {
			return err
		}
		for _, strike := range group.Facts {
			if !seen[strike.Account] {
				seen[strike.Account] = true
				woken = append(woken, strike.Account)
			}
		}
	}
	for _, account := range woken {
		u.streams.Publish(account)
	}
	return nil
}

func (u *UseCase) record(ctx context.Context, group days.Group[string, flagmissions.Strike]) error {
	flag := group.Key
	if _, err := u.briefing.Of(ctx, group.Day, flag); err != nil {
		return fmt.Errorf("failed to read %s's mission: %w", flag, err)
	}

	return u.transactioner.InTx(ctx, func(ctx context.Context) error {
		var done []flagmissions.Done
		if err := u.flagDays.ChangeDay(ctx, group.Day, flag, func(day flagmissions.Day) flagmissions.Day {
			for _, strike := range group.Facts {
				var finished []flagmissions.Done
				day, finished = day.WithStrike(strike)
				done = append(done, finished...)
			}
			return day
		}); err != nil {
			return fmt.Errorf("failed to count the shields broken on %s's day: %w", flag, err)
		}
		if len(done) == 0 {
			return nil
		}
		told, err := flagmessage.Done(flag, done)
		if err != nil {
			return fmt.Errorf("failed to tell %s's mission done: %w", flag, err)
		}
		return u.outbox.Record(ctx, append(scoringgrant.Of(flag, done), told...)...) //nolint:wrapcheck // the outbox named it.
	})
}
