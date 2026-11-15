package backfill_days_played_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
)

type Stats interface {
	StatsAfter(ctx context.Context, after accounts.AccountID, limit int) ([]stats.Stats, error)
	Record(ctx context.Context, account accounts.AccountID, change func(stats.Stats) stats.Stats) error
}

type PlayDays interface {
	PlayDays(ctx context.Context, accounts []accounts.AccountID) (map[accounts.AccountID][]stats.Day, error)
}

type Executor interface {
	Execute(ctx context.Context, dryRun bool) (Backfilled, error)
}

type Backfilled struct {
	Accounts int
	Changed  int
}

// Planet's GetPlayDays takes 500 accounts at most.
const pageSize = 500

type UseCase struct {
	stats    Stats
	playDays PlayDays
}

var _ Executor = (*UseCase)(nil)

func New(stats Stats, playDays PlayDays) *UseCase {
	return &UseCase{stats: stats, playDays: playDays}
}

// Execute counts every account's days played again from the days planet's ledger holds a take on. An account the page
// shows unchanged is not written; the others are, under the stats' lock, so a take counted live meanwhile is counted
// once. A second run changes nothing, and a failed one is simply run again.
func (u *UseCase) Execute(ctx context.Context, dryRun bool) (Backfilled, error) {
	var (
		backfilled Backfilled
		after      accounts.AccountID
	)
	for {
		page, err := u.stats.StatsAfter(ctx, after, pageSize)
		if err != nil {
			return backfilled, fmt.Errorf("failed to read a page of stats: %w", err)
		}
		if len(page) == 0 {
			return backfilled, nil
		}

		played, err := u.playDays.PlayDays(ctx, stats.AccountsOf(page))
		if err != nil {
			return backfilled, fmt.Errorf("failed to read the days played of a page: %w", err)
		}

		for _, counted := range page {
			backfilled.Accounts++
			days := played[counted.Account()]
			if counted.WithPlayDays(days).DaysPlayed() == counted.DaysPlayed() {
				continue
			}
			backfilled.Changed++
			if dryRun {
				continue
			}
			if err := u.stats.Record(ctx, counted.Account(), stats.PlayDays(days)); err != nil {
				return backfilled, fmt.Errorf("failed to record the days played of an account: %w", err)
			}
		}

		if len(page) < pageSize {
			return backfilled, nil
		}
		after = page[len(page)-1].Account()
	}
}
