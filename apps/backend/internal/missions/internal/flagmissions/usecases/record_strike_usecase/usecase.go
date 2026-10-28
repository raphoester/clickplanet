package record_strike_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions/scoringgrant"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type Briefing interface {
	Of(ctx context.Context, day missions.Day, flag string) (flagmissions.Day, error)
}

type Days interface {
	ChangeDay(ctx context.Context, day missions.Day, flag string, change func(flagmissions.Day) flagmissions.Day) error
}

func New(
	calendar missions.Calendar, briefing Briefing, days Days, transactioner cpatomic.Transactioner, outbox cpatomic.Outbox,
) *UseCase {
	return &UseCase{calendar: calendar, briefing: briefing, days: days, transactioner: transactioner, outbox: outbox}
}

type UseCase struct {
	calendar      missions.Calendar
	briefing      Briefing
	days          Days
	transactioner cpatomic.Transactioner
	outbox        cpatomic.Outbox
}

func (u *UseCase) Execute(ctx context.Context, strike flagmissions.Strike) error {
	day := u.calendar.DayOf(strike.At)
	if _, err := u.briefing.Of(ctx, day, strike.Flag); err != nil {
		return fmt.Errorf("failed to read %s's mission: %w", strike.Flag, err)
	}

	return u.transactioner.InTx(ctx, func(ctx context.Context) error {
		var done []flagmissions.Done
		if err := u.days.ChangeDay(ctx, day, strike.Flag, func(current flagmissions.Day) flagmissions.Day {
			next, finished := current.WithStrike(strike)
			done = finished
			return next
		}); err != nil {
			return fmt.Errorf("failed to count the shield broken on %s's day: %w", strike.Flag, err)
		}
		if len(done) == 0 {
			return nil
		}
		return u.outbox.Record(ctx, scoringgrant.Of(strike.Flag, done, strike.At)...) //nolint:wrapcheck // the outbox named it.
	})
}
