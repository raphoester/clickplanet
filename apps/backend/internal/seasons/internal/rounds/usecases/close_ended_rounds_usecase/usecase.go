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
	Open(ctx context.Context, round rounds.Round) error
}

type Closer interface {
	Execute(ctx context.Context, round rounds.Round, snapshot rounds.Snapshot) (rounds.Closed, error)
}

type Executor interface {
	Execute(ctx context.Context) (Report, error)
}

type Report struct {
	Closed []rounds.Closed
	ReadAt time.Time
	Next   time.Time
}

const timeout = 30 * time.Second

func New(territory Territory, store Rounds, closer Closer, seasons calendar.Calendar, clock cptime.Clock) *UseCase {
	return &UseCase{territory: territory, store: store, closer: closer, seasons: seasons, clock: clock}
}

type UseCase struct {
	territory Territory
	store     Rounds
	closer    Closer
	seasons   calendar.Calendar
	clock     cptime.Clock
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context) (Report, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	now := u.clock.Now()
	closed, closing := u.close(ctx, now)
	next, opening := u.open(ctx, now)
	return Report{Closed: closed, ReadAt: now, Next: next}, errors.Join(closing, opening)
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

func (u *UseCase) open(ctx context.Context, now time.Time) (time.Time, error) {
	round, ok := rounds.Current(u.seasons, now)
	if !ok {
		return time.Time{}, nil
	}
	if err := u.store.Open(ctx, round); err != nil {
		return time.Time{}, fmt.Errorf("failed to open the round in progress: %w", err)
	}
	return round.EndsAt, nil
}
