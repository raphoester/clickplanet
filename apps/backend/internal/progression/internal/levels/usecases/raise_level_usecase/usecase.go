package raise_level_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Stats interface {
	Stats(ctx context.Context, account accounts.AccountID) (stats.Stats, error)
}

type Accounts interface {
	Account(ctx context.Context, account accounts.AccountID) (accounts.Account, error)
}

type Levels interface {
	Levels(ctx context.Context, accounts []accounts.AccountID) (levels.ByAccount, error)
	Raise(ctx context.Context, account accounts.AccountID, level levels.Level, at time.Time) (levels.Level, error)
}

// Feed tells the account's open streams: the XP each fact gave, and each level reached.
type Feed interface {
	Count(account accounts.AccountID, earned levels.Earned, standing levels.Standing)
	Publish(account accounts.AccountID, reached levels.Standing)
}

type UseCase struct {
	stats    Stats
	accounts Accounts
	levels   Levels
	feed     Feed
	clock    cptime.Clock
}

func New(stats Stats, accounts Accounts, levels Levels, feed Feed, clock cptime.Clock) *UseCase {
	return &UseCase{stats: stats, accounts: accounts, levels: levels, feed: feed, clock: clock}
}

// Execute answers the level it raised the account to, 0 when it raised none.
func (u *UseCase) Execute(ctx context.Context, account accounts.AccountID) (levels.Level, error) {
	counted, err := u.stats.Stats(ctx, account)
	if errors.Is(err, stats.ErrNoStats) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to read the stats: %w", err)
	}

	kept, err := u.levels.Levels(ctx, []accounts.AccountID{account})
	if err != nil {
		return 0, fmt.Errorf("failed to read the level kept: %w", err)
	}
	scale := levels.Scale{}
	reach := scale.Reach(counted)
	if reach <= kept[account] {
		// A level is kept, so the account is linked: it hears the XP the fact gave.
		u.feed.Count(account, scale.EarnedOf(counted), scale.StandingAt(kept[account], counted))
		return 0, nil
	}

	known, err := u.accounts.Account(ctx, account)
	if err != nil {
		return 0, fmt.Errorf("failed to ask player about the account: %w", err)
	}
	if scale.StandingOf(known, counted).Empty() {
		return 0, nil
	}

	previous, err := u.levels.Raise(ctx, account, reach, u.clock.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to raise the level: %w", err)
	}
	if previous >= reach {
		return 0, nil
	}
	reached := scale.StandingAt(reach, counted)
	u.feed.Count(account, scale.EarnedOf(counted), reached)
	u.feed.Publish(account, reached)
	return reach, nil
}
