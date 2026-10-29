package record_flag_take_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions/scoringgrant"
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

func New(
	calendar Calendar, briefing Briefing, flagDays FlagDays, transactioner cpatomic.Transactioner, outbox cpatomic.Outbox,
) *UseCase {
	return &UseCase{calendar: calendar, briefing: briefing, flagDays: flagDays, transactioner: transactioner, outbox: outbox}
}

type UseCase struct {
	calendar      Calendar
	briefing      Briefing
	flagDays      FlagDays
	transactioner cpatomic.Transactioner
	outbox        cpatomic.Outbox
}

func (u *UseCase) Execute(ctx context.Context, take flagmissions.Take) error {
	day, err := u.calendar.DayAt(ctx, take.At)
	if errors.Is(err, days.ErrNoDay) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read the day of the take: %w", err)
	}
	if _, err := u.briefing.Of(ctx, day, take.Flag); err != nil {
		return fmt.Errorf("failed to read %s's mission: %w", take.Flag, err)
	}

	return u.transactioner.InTx(ctx, func(ctx context.Context) error {
		var done []flagmissions.Done
		if err := u.flagDays.ChangeDay(ctx, day, take.Flag, func(current flagmissions.Day) flagmissions.Day {
			next, finished := current.WithTake(take)
			done = finished
			return next
		}); err != nil {
			return fmt.Errorf("failed to count the take on %s's day: %w", take.Flag, err)
		}
		if len(done) == 0 {
			return nil
		}
		return u.outbox.Record(ctx, scoringgrant.Of(take.Flag, done, take.At)...) //nolint:wrapcheck // the outbox named it.
	})
}
