package forgive_missed_day_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/usecases/record_stats_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Stats interface {
	Stats(ctx context.Context, account accounts.AccountID) (stats.Stats, error)
}

type PlayDays interface {
	PlayDays(ctx context.Context, accounts []accounts.AccountID) (map[accounts.AccountID][]stats.Day, error)
}

type Recorder interface {
	Execute(ctx context.Context, in record_stats_usecase.In) error
}

type Executor interface {
	Execute(ctx context.Context, in In) (Forgiven, error)
}

type In struct {
	Account accounts.AccountID
	Day     stats.Day
	DryRun  bool
}

type Forgiven struct {
	Before stats.Stats
	After  stats.Stats
}

type UseCase struct {
	stats    Stats
	playDays PlayDays
	record   Recorder
	clock    cptime.Clock
}

var _ Executor = (*UseCase)(nil)

func New(stats Stats, playDays PlayDays, record Recorder, clock cptime.Clock) *UseCase {
	return &UseCase{stats: stats, playDays: playDays, record: record, clock: clock}
}

// Execute asks the rule with the stats read and planet's days, then records it under the stats' lock, where it is asked
// again, so the answer reads the stats back.
func (u *UseCase) Execute(ctx context.Context, in In) (Forgiven, error) {
	before, err := u.stats.Stats(ctx, in.Account)
	if err != nil {
		return Forgiven{}, fmt.Errorf("failed to read the account's stats: %w", err)
	}
	played, err := u.playDays.PlayDays(ctx, []accounts.AccountID{in.Account})
	if err != nil {
		return Forgiven{Before: before}, fmt.Errorf("failed to read the days the account played: %w", err)
	}

	today := stats.DayOf(u.clock.Now())
	after, err := before.WithForgivenDay(in.Day, played[in.Account], today)
	if err != nil {
		return Forgiven{Before: before}, fmt.Errorf("refused to forgive %s: %w", in.Day, err)
	}
	if in.DryRun {
		return Forgiven{Before: before, After: after}, nil
	}

	if err := u.record.Execute(ctx, record_stats_usecase.In{
		Account: in.Account, Change: stats.ForgivenDay(in.Day, played[in.Account], today),
	}); err != nil {
		return Forgiven{Before: before}, fmt.Errorf("failed to record the forgiven day: %w", err)
	}
	after, err = u.stats.Stats(ctx, in.Account)
	if err != nil {
		return Forgiven{Before: before}, fmt.Errorf("failed to read the account's stats back: %w", err)
	}
	return Forgiven{Before: before, After: after}, nil
}
