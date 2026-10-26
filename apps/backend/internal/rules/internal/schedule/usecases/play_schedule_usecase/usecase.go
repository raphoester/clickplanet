package play_schedule_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/rules/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Entries interface {
	Unended(ctx context.Context, at time.Time) ([]schedule.Entry, error)
}

type Commands interface {
	Play(ctx context.Context, turn schedule.Turn, decidedAt time.Time) error
}

// Execute answers the turn at now, and told is true when it commanded it.
type Executor interface {
	Execute(ctx context.Context) (turn schedule.Turn, told bool, err error)
}

func New(entries Entries, commands Commands, clock cptime.Clock) *UseCase {
	return &UseCase{entries: entries, commands: commands, clock: clock}
}

// UseCase commands the turn once it changes, and once after each boot, when it remembers none. A command Redis did not
// take is sent again at the next Execute.
type UseCase struct {
	entries  Entries
	commands Commands
	clock    cptime.Clock

	told bool
	last schedule.Turn
}

func (u *UseCase) Execute(ctx context.Context) (schedule.Turn, bool, error) {
	now := u.clock.Now()
	entries, err := u.entries.Unended(ctx, now)
	if err != nil {
		return schedule.Turn{}, false, fmt.Errorf("could not read the schedule: %w", err)
	}

	turn := schedule.New(entries...).TurnAt(now)
	if u.told && turn == u.last {
		return turn, false, nil
	}
	if err := u.commands.Play(ctx, turn, now); err != nil {
		return turn, false, err //nolint:wrapcheck // the adapter names the set.
	}
	u.told, u.last = true, turn

	return turn, true, nil
}
