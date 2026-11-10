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
	// Play tells planet and the antibot the turn's set.
	Play(ctx context.Context, turn schedule.Turn, decidedAt time.Time) error
	// PlayAntibot tells the antibot alone: planet's set waits for its battle's map.
	PlayAntibot(ctx context.Context, turn schedule.Turn, decidedAt time.Time) error
}

type Battles interface {
	// SetWaits is true while planet must not play the turn's set yet: its battle is not judged, or its arena not opened.
	SetWaits(ctx context.Context, turn schedule.Turn, at time.Time) (bool, error)
}

// Execute answers the turn at now, and told is true when it commanded a new one.
type Executor interface {
	Execute(ctx context.Context) (turn schedule.Turn, told bool, err error)
}

func New(
	entries Entries, seasons Seasons, battles Battles, commands Commands, clock cptime.Clock, again time.Duration,
) *UseCase {
	return &UseCase{entries: entries, seasons: seasons, battles: battles, commands: commands, clock: clock, again: again}
}

// UseCase commands the turn once it changes, and once after each boot, when it remembers none, decided then. It sends
// the turn again each `again` with that same decision time: a module that plays it already changes nothing, and one that
// refused its set plays it once the set is made. A command Redis did not take is sent again at the next Execute.
//
// Planet plays a battle's set only once the battle took its map (Battles.SetWaits): until then the antibot alone is told,
// and planet keeps the set before. Planet is told as soon as the wait ends, with the turn's decision time.
type UseCase struct {
	entries  Entries
	seasons  Seasons
	battles  Battles
	commands Commands
	clock    cptime.Clock
	again    time.Duration

	told       bool
	last       schedule.Turn
	decidedAt  time.Time
	sentAt     time.Time
	planetTold bool
}

// Execute answers the turn at now, and told is true when the turn is new, this is the first since the boot, or planet is
// told the turn's set once its wait ended; a turn sent again is not told.
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

	fresh := !u.told || turn != u.last
	if !fresh && u.planetTold && now.Sub(u.sentAt) < u.again {
		return turn, false, nil
	}
	waits, err := u.planetWaits(ctx, turn, fresh, now)
	if err != nil {
		return turn, false, err
	}
	switch {
	case fresh:
		if err := u.send(ctx, turn, now, waits); err != nil {
			return turn, false, err
		}
		u.told, u.last, u.decidedAt, u.sentAt, u.planetTold = true, turn, now, now, !waits
		return turn, true, nil
	case !u.planetTold && !waits:
		if err := u.send(ctx, turn, u.decidedAt, false); err != nil {
			return turn, false, err
		}
		u.sentAt, u.planetTold = now, true
		return turn, true, nil
	case now.Sub(u.sentAt) >= u.again:
		if err := u.send(ctx, turn, u.decidedAt, waits); err != nil {
			return turn, false, err
		}
		u.sentAt = now
	}
	return turn, false, nil
}

// planetWaits asks only while planet has not been told the turn's set: once it plays it, nothing takes it back.
func (u *UseCase) planetWaits(ctx context.Context, turn schedule.Turn, fresh bool, now time.Time) (bool, error) {
	if (!fresh && u.planetTold) || turn.TakeSnapshot().Kind != schedule.Battle {
		return false, nil
	}
	waits, err := u.battles.SetWaits(ctx, turn, now)
	if err != nil {
		return false, fmt.Errorf("could not read whether the battle's set waits: %w", err)
	}
	return waits, nil
}

func (u *UseCase) send(ctx context.Context, turn schedule.Turn, decidedAt time.Time, planetWaits bool) error {
	if planetWaits {
		return u.commands.PlayAntibot(ctx, turn, decidedAt) //nolint:wrapcheck // the adapter names the set.
	}
	return u.commands.Play(ctx, turn, decidedAt) //nolint:wrapcheck // the adapter names the set.
}
