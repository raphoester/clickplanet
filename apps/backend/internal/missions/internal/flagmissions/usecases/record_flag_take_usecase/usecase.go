package record_flag_take_usecase

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

func (u *UseCase) Execute(ctx context.Context, take missions.Take) error {
	day := u.calendar.DayOf(take.At)
	if _, err := u.briefing.Of(ctx, day, take.Flag); err != nil {
		return fmt.Errorf("failed to read %s's mission: %w", take.Flag, err)
	}

	return u.transactioner.InTx(ctx, func(ctx context.Context) error {
		var done []flagmissions.Done
		if err := u.days.ChangeDay(ctx, day, take.Flag, func(current flagmissions.Day) flagmissions.Day {
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
