package take_snapshot_usecase

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
	RecordSnapshot(ctx context.Context, round rounds.Round, snapshot rounds.Snapshot) error
	Unclosed(ctx context.Context, endedBy time.Time) ([]rounds.Round, error)
}

type Closer interface {
	Execute(ctx context.Context, round rounds.Round) (rounds.Closed, error)
}

type Executor interface {
	Execute(ctx context.Context) ([]rounds.Closed, error)
}

type Config struct {
	Interval time.Duration
}

const (
	defaultInterval = time.Minute
	timeout         = 30 * time.Second
)

func (c Config) WithDefaults() Config {
	if c.Interval <= 0 {
		c.Interval = defaultInterval
	}
	return c
}

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

func (u *UseCase) Execute(ctx context.Context) ([]rounds.Closed, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	now := u.clock.Now()
	closed, closing := u.close(ctx, now)
	return closed, errors.Join(closing, u.count(ctx, now))
}

func (u *UseCase) close(ctx context.Context, now time.Time) ([]rounds.Closed, error) {
	ended, err := u.store.Unclosed(ctx, now)
	if err != nil {
		return nil, fmt.Errorf("failed to read the rounds that ended: %w", err)
	}

	closed := make([]rounds.Closed, 0, len(ended))
	for _, round := range ended {
		closing, err := u.closer.Execute(ctx, round)
		if err != nil {
			return closed, fmt.Errorf("failed to close the round: %w", err)
		}
		closed = append(closed, closing)
	}
	return closed, nil
}

func (u *UseCase) count(ctx context.Context, now time.Time) error {
	round, ok := rounds.Current(u.seasons, now)
	if !ok {
		return nil
	}

	snapshot, err := u.territory.Snapshot(ctx)
	if err != nil {
		return fmt.Errorf("failed to count what each country holds: %w", err)
	}
	if err := u.store.RecordSnapshot(ctx, round, snapshot); err != nil {
		return fmt.Errorf("failed to record the snapshot: %w", err)
	}
	return nil
}
