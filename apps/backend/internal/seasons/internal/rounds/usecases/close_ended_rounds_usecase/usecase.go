package close_ended_rounds_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Territory interface {
	Snapshot(ctx context.Context) (rounds.Snapshot, error)
}

type Rounds interface {
	Unclosed(ctx context.Context, endedBy time.Time) ([]rounds.Round, error)
	FirstEnd(ctx context.Context) (time.Time, bool, error)
	Ended(ctx context.Context, season calendar.Number) (bool, error)
	Open(ctx context.Context, round rounds.Round) error
}

type Closer interface {
	Execute(ctx context.Context, round rounds.Round, snapshot rounds.Snapshot) (rounds.Closed, error)
}

type Ender interface {
	Execute(ctx context.Context, season calendar.Season) (rounds.SeasonEnd, bool, error)
}

type Executor interface {
	Execute(ctx context.Context) (Report, error)
}

type Report struct {
	Closed []rounds.Closed
	Ended  []rounds.SeasonEnd
	ReadAt time.Time
	// Next is when to run again: the first end of a round or of the season. Zero when nothing is left to end.
	Next time.Time
}

const timeout = 30 * time.Second

func New(territory Territory, store Rounds, closer Closer, ender Ender, seasons calendar.Calendar, clock cptime.Clock) *UseCase {
	return &UseCase{territory: territory, store: store, closer: closer, ender: ender, seasons: seasons, clock: clock}
}

// UseCase closes every round that ended, from one read of the map, then ends every season whose end is past, then opens
// the day in progress.
type UseCase struct {
	territory Territory
	store     Rounds
	closer    Closer
	ender     Ender
	seasons   calendar.Calendar
	clock     cptime.Clock
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context) (Report, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	now := u.clock.Now()
	report := Report{ReadAt: now}
	var closing, ending error
	report.Closed, closing = u.close(ctx, now)
	if closing == nil {
		report.Ended, ending = u.end(ctx, now)
	}
	opening := u.open(ctx, now)
	next, timing := u.next(ctx, now)
	report.Next = next
	return report, errors.Join(closing, ending, opening, timing)
}

func (u *UseCase) close(ctx context.Context, now time.Time) ([]rounds.Closed, error) {
	ended, err := u.store.Unclosed(ctx, now)
	if err != nil {
		return nil, fmt.Errorf("failed to read the rounds that ended: %w", err)
	}
	if len(ended) == 0 {
		return nil, nil
	}

	end, err := u.territory.Snapshot(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read what each country holds at the end of the round: %w", err)
	}
	closed := make([]rounds.Closed, 0, len(ended))
	for _, round := range ended {
		closing, err := u.closer.Execute(ctx, round, end)
		if err != nil {
			return closed, fmt.Errorf("failed to close the round: %w", err)
		}
		closed = append(closed, closing)
	}
	return closed, nil
}

// end runs once every round that ended is closed, so each season's table holds all of its rounds.
func (u *UseCase) end(ctx context.Context, now time.Time) ([]rounds.SeasonEnd, error) {
	var ends []rounds.SeasonEnd
	for _, season := range u.seasons.Ended(now) {
		done, err := u.store.Ended(ctx, season.Number)
		if err != nil {
			return ends, fmt.Errorf("failed to read whether season %d ended: %w", season.Number, err)
		}
		if done {
			continue
		}
		end, ended, err := u.ender.Execute(ctx, season)
		if err != nil {
			return ends, fmt.Errorf("failed to end season %d: %w", season.Number, err)
		}
		if ended {
			ends = append(ends, end)
		}
	}
	return ends, nil
}

func (u *UseCase) open(ctx context.Context, now time.Time) error {
	round, ok := rounds.Day(u.seasons, now)
	if !ok {
		return nil
	}
	if err := u.store.Open(ctx, round); err != nil {
		return fmt.Errorf("failed to open the day in progress: %w", err)
	}
	return nil
}

func (u *UseCase) next(ctx context.Context, now time.Time) (time.Time, error) {
	var next time.Time
	if season, ok := u.seasons.Current(now); ok {
		next = season.EndsAt
	}
	first, found, err := u.store.FirstEnd(ctx)
	if err != nil {
		return next, fmt.Errorf("failed to read when the next round ends: %w", err)
	}
	if found && (next.IsZero() || first.Before(next)) {
		next = first
	}
	return next, nil
}
