package take_census_usecase

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
	Census(ctx context.Context) (rounds.Census, error)
}

type Rounds interface {
	RecordCensus(ctx context.Context, round rounds.Round, census rounds.Census) error
	Unclosed(ctx context.Context, endedBy time.Time) ([]rounds.Round, error)
	Held(ctx context.Context, round rounds.Round) (map[rounds.Country]uint64, error)
	Close(ctx context.Context, round rounds.Round, results []rounds.Result) error
}

type Executor interface {
	Execute(ctx context.Context) ([]rounds.Round, error)
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

func New(territory Territory, store Rounds, seasons calendar.Calendar, clock cptime.Clock) *UseCase {
	return &UseCase{territory: territory, store: store, seasons: seasons, clock: clock}
}

type UseCase struct {
	territory Territory
	store     Rounds
	seasons   calendar.Calendar
	clock     cptime.Clock
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context) ([]rounds.Round, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	now := u.clock.Now()
	closed, closing := u.close(ctx, now)
	return closed, errors.Join(closing, u.count(ctx, now))
}

func (u *UseCase) close(ctx context.Context, now time.Time) ([]rounds.Round, error) {
	ended, err := u.store.Unclosed(ctx, now)
	if err != nil {
		return nil, fmt.Errorf("failed to read the rounds that ended: %w", err)
	}

	closed := make([]rounds.Round, 0, len(ended))
	for _, round := range ended {
		held, err := u.store.Held(ctx, round)
		if err != nil {
			return closed, fmt.Errorf("failed to read what each country held in the round: %w", err)
		}
		if err := u.store.Close(ctx, round, round.Results(held)); err != nil {
			return closed, fmt.Errorf("failed to close the round: %w", err)
		}
		closed = append(closed, round)
	}
	return closed, nil
}

func (u *UseCase) count(ctx context.Context, now time.Time) error {
	round, ok := rounds.Current(u.seasons, now)
	if !ok {
		return nil
	}

	census, err := u.territory.Census(ctx)
	if err != nil {
		return fmt.Errorf("failed to count what each country holds: %w", err)
	}
	if err := u.store.RecordCensus(ctx, round, census); err != nil {
		return fmt.Errorf("failed to record the census: %w", err)
	}
	return nil
}
