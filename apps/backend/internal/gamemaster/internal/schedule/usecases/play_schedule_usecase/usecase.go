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
	// Play tells planet and the antibot the turn's set: nil once both play it.
	Play(ctx context.Context, turn schedule.Turn, decidedAt time.Time) error
	// PlayAntibot tells the antibot alone: planet's set waits for its battle's map.
	PlayAntibot(ctx context.Context, turn schedule.Turn, decidedAt time.Time) error
}

type Battles interface {
	// SetWaits is true while planet must not play the turn's set yet: its battle is not judged, or its arena not opened.
	SetWaits(ctx context.Context, turn schedule.Turn, at time.Time) (bool, error)
}

// Execute answers the turn at now, and told is true when a module played its set at this Execute.
type Executor interface {
	Execute(ctx context.Context) (turn schedule.Turn, told bool, err error)
}

func New(
	entries Entries, seasons Seasons, battles Battles, commands Commands, clock cptime.Clock, again time.Duration,
) *UseCase {
	return &UseCase{entries: entries, seasons: seasons, battles: battles, commands: commands, clock: clock, again: again}
}

// played is how far the turn's set is played.
type played uint8

const (
	byNone played = iota
	byAntibot
	byBoth
)

// UseCase tells the turn once it changes, and once after each boot, decided then, until both modules answer that they
// play it. A call not answered, or a set refused, is made again after `again` with the same decision time: a module
// that refused a set plays it once the operator makes it.
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

	told      bool
	last      schedule.Turn
	decidedAt time.Time
	played    played
	failedAt  time.Time
}

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

	if !u.told || turn != u.last {
		u.told, u.last, u.decidedAt, u.played, u.failedAt = true, turn, now, byNone, time.Time{}
	}
	if u.played == byBoth || (!u.failedAt.IsZero() && now.Sub(u.failedAt) < u.again) {
		return turn, false, nil
	}
	waits, err := u.planetWaits(ctx, turn, now)
	if err != nil {
		return turn, false, err
	}
	if waits && u.played == byAntibot {
		return turn, false, nil
	}
	if err := u.send(ctx, turn, waits); err != nil {
		u.failedAt = now
		return turn, false, err
	}
	u.played, u.failedAt = byBoth, time.Time{}
	if waits {
		u.played = byAntibot
	}
	return turn, true, nil
}

func (u *UseCase) planetWaits(ctx context.Context, turn schedule.Turn, now time.Time) (bool, error) {
	if turn.TakeSnapshot().Kind != schedule.Battle {
		return false, nil
	}
	waits, err := u.battles.SetWaits(ctx, turn, now)
	if err != nil {
		return false, fmt.Errorf("could not read whether the battle's set waits: %w", err)
	}
	return waits, nil
}

func (u *UseCase) send(ctx context.Context, turn schedule.Turn, planetWaits bool) error {
	if planetWaits {
		return u.commands.PlayAntibot(ctx, turn, u.decidedAt) //nolint:wrapcheck // the adapter names the set.
	}
	return u.commands.Play(ctx, turn, u.decidedAt) //nolint:wrapcheck // the adapter names the set.
}
