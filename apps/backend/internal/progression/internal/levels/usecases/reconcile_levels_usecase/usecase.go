package reconcile_levels_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Stats interface {
	StatsAfter(ctx context.Context, after accounts.AccountID, limit int) ([]stats.Stats, error)
}

type Accounts interface {
	Accounts(ctx context.Context, accounts []accounts.AccountID) (map[accounts.AccountID]accounts.Account, error)
}

type Levels interface {
	Levels(ctx context.Context, accounts []accounts.AccountID) (levels.ByAccount, error)
	RaiseAll(ctx context.Context, levels levels.ByAccount, at time.Time) error
	Set(ctx context.Context, levels levels.ByAccount, at time.Time) error
	Forget(ctx context.Context, accounts ...accounts.AccountID) error
}

type Executor interface {
	Execute(ctx context.Context, dryRun bool) (Reconciled, error)
}

type Reconciled struct {
	Raised  int
	Lowered int
	Removed int
}

const pageSize = 500

type UseCase struct {
	stats      Stats
	accounts   Accounts
	levels     Levels
	clock      cptime.Clock
	visibility levels.Visibility
}

var _ Executor = (*UseCase)(nil)

func New(stats Stats, accounts Accounts, levels Levels, clock cptime.Clock, visibility levels.Visibility) *UseCase {
	return &UseCase{stats: stats, accounts: accounts, levels: levels, clock: clock, visibility: visibility}
}

func (u *UseCase) Execute(ctx context.Context, dryRun bool) (Reconciled, error) {
	at := u.clock.Now()
	var (
		reconciled Reconciled
		after      accounts.AccountID
	)
	for {
		page, err := u.stats.StatsAfter(ctx, after, pageSize)
		if err != nil {
			return reconciled, fmt.Errorf("failed to read a page of stats: %w", err)
		}
		if len(page) == 0 {
			return reconciled, nil
		}

		ids := stats.AccountsOf(page)
		known, err := u.accounts.Accounts(ctx, ids)
		if err != nil {
			return reconciled, fmt.Errorf("failed to ask player about a page of accounts: %w", err)
		}
		kept, err := u.levels.Levels(ctx, ids)
		if err != nil {
			return reconciled, fmt.Errorf("failed to read the levels of a page: %w", err)
		}

		reconciliation := levels.Scale{}.ReconciliationOf(page, known, kept, u.visibility)
		if !dryRun {
			if err := u.write(ctx, reconciliation, at); err != nil {
				return reconciled, err
			}
		}
		reconciled.Raised += len(reconciliation.Raised())
		reconciled.Lowered += len(reconciliation.Lowered())
		reconciled.Removed += len(reconciliation.Removed())

		if len(page) < pageSize {
			return reconciled, nil
		}
		after = page[len(page)-1].Account()
	}
}

func (u *UseCase) write(ctx context.Context, reconciliation levels.Reconciliation, at time.Time) error {
	if err := u.levels.RaiseAll(ctx, reconciliation.Raised(), at); err != nil {
		return fmt.Errorf("failed to raise the levels: %w", err)
	}
	if err := u.levels.Set(ctx, reconciliation.Lowered(), at); err != nil {
		return fmt.Errorf("failed to lower the levels: %w", err)
	}
	if err := u.levels.Forget(ctx, reconciliation.Removed()...); err != nil {
		return fmt.Errorf("failed to remove the guests' levels: %w", err)
	}
	return nil
}
