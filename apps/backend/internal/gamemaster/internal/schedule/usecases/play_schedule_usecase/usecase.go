package play_schedule_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Entries interface {
	Unended(ctx context.Context, at time.Time) ([]schedule.Entry, error)
}

type Seasons interface {
	Calendar(ctx context.Context) (season.Calendar, error)
}

type Commands interface {
	Play(ctx context.Context, turn schedule.Turn, decidedAt time.Time) error
}

// Execute answers the turn at now, and told is true when it commanded a new one.
type Executor interface {
	Execute(ctx context.Context) (turn schedule.Turn, told bool, err error)
}

func New(entries Entries, seasons Seasons, commands Commands, clock cptime.Clock, again time.Duration) *UseCase {
	return &UseCase{entries: entries, seasons: seasons, commands: commands, clock: clock, again: again}
}

// UseCase commands the turn once it changes, and once after each boot, when it remembers none, decided then. It sends
// the turn again each `again` with that same decision time: a module that plays it already changes nothing, and one that
// refused its set plays it once the set is made. A command Redis did not take is sent again at the next Execute.
type UseCase struct {
	entries  Entries
	seasons  Seasons
	commands Commands
	clock    cptime.Clock
	again    time.Duration

	told      bool
	last      schedule.Turn
	decidedAt time.Time
	sentAt    time.Time
}

// Execute answers the turn at now, and told is true when the turn is new or this is the first since the boot; a turn
// sent again is not told.
func (u *UseCase) Execute(ctx context.Context) (schedule.Turn, bool, error) {
	now := u.clock.Now()
	entries, err := u.entries.Unended(ctx, now)
	if err != nil {
		return schedule.Turn{}, false, fmt.Errorf("could not read the schedule: %w", err)
	}

	calendar, err := u.seasons.Calendar(ctx)
	if err != nil {
		return schedule.Turn{}, false, fmt.Errorf("could not read the seasons: %w", err)
	}
	turn := schedule.New(entries...).With(calendar.Finales()...).TurnAt(now)
	if u.told && turn == u.last {
		if now.Sub(u.sentAt) < u.again {
			return turn, false, nil
		}
		if err := u.commands.Play(ctx, turn, u.decidedAt); err != nil {
			return turn, false, err //nolint:wrapcheck // the adapter names the set.
		}
		u.sentAt = now
		return turn, false, nil
	}
	if err := u.commands.Play(ctx, turn, now); err != nil {
		return turn, false, err //nolint:wrapcheck // the adapter names the set.
	}
	u.told, u.last, u.decidedAt, u.sentAt = true, turn, now, now

	return turn, true, nil
}
