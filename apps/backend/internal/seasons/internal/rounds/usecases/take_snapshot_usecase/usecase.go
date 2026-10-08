package take_snapshot_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
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
	Held(ctx context.Context, round rounds.Round) (map[rounds.Country]uint64, error)
	Number(ctx context.Context, round rounds.Round) (uint32, error)
	Close(ctx context.Context, round rounds.Round, results []rounds.Result) error
}

type Publisher interface {
	Publish(event proto.Message)
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

func New(territory Territory, store Rounds, events Publisher, seasons calendar.Calendar, clock cptime.Clock) *UseCase {
	return &UseCase{territory: territory, store: store, events: events, seasons: seasons, clock: clock}
}

type UseCase struct {
	territory Territory
	store     Rounds
	events    Publisher
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
		held, err := u.store.Held(ctx, round)
		if err != nil {
			return closed, fmt.Errorf("failed to read what each country held in the round: %w", err)
		}
		number, err := u.store.Number(ctx, round)
		if err != nil {
			return closed, fmt.Errorf("failed to number the round: %w", err)
		}
		closing := round.Closed(number, held)
		if err := u.store.Close(ctx, round, closing.Results); err != nil {
			return closed, fmt.Errorf("failed to close the round: %w", err)
		}
		u.events.Publish(eventOf(closing))
		closed = append(closed, closing)
	}
	return closed, nil
}

func eventOf(closed rounds.Closed) *seasonsv1.RoundClosed {
	results := make([]*seasonsv1.RoundResult, 0, len(closed.Results))
	for _, result := range closed.Results {
		results = append(results, &seasonsv1.RoundResult{
			Country: string(result.Country),
			Rank:    result.Rank,
			Points:  result.Points,
		})
	}
	return &seasonsv1.RoundClosed{
		Season:  uint32(closed.Round.Season),
		Number:  closed.Number,
		Finale:  closed.Round.Finale,
		EndedAt: timestamppb.New(closed.Round.EndsAt),
		Results: results,
	}
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
