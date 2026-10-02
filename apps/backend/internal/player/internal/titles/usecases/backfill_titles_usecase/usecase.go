package backfill_titles_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Stats interface {
	StatsAfter(ctx context.Context, after players.AccountID, limit int) ([]players.Stats, error)
}

type Accounts interface {
	CreationDates(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]time.Time, error)
}

type Titles interface {
	Grant(ctx context.Context, grants titles.Grants, at time.Time) error
}

type Executor interface {
	Execute(ctx context.Context) (Backfill, error)
}

type Backfill struct {
	Accounts int
}

const pageSize = 500

type UseCase struct {
	stats    Stats
	accounts Accounts
	titles   Titles
	catalog  titles.Catalog
	clock    cptime.Clock
}

var _ Executor = (*UseCase)(nil)

func New(stats Stats, accounts Accounts, titles Titles, catalog titles.Catalog, clock cptime.Clock) *UseCase {
	return &UseCase{stats: stats, accounts: accounts, titles: titles, catalog: catalog, clock: clock}
}

func (u *UseCase) Execute(ctx context.Context) (Backfill, error) {
	at := u.clock.Now()
	var (
		backfill Backfill
		after    players.AccountID
	)
	for {
		page, err := u.stats.StatsAfter(ctx, after, pageSize)
		if err != nil {
			return backfill, fmt.Errorf("failed to read a page of stats: %w", err)
		}

		created, err := u.accounts.CreationDates(ctx, titles.AccountsOf(page))
		if err != nil {
			return backfill, fmt.Errorf("failed to ask when a page of accounts was made: %w", err)
		}

		grants := u.catalog.GrantsFor(titles.CareersOf(page, created))
		if err := u.titles.Grant(ctx, grants, at); err != nil {
			return backfill, fmt.Errorf("failed to grant the titles: %w", err)
		}
		backfill.Accounts += len(grants)

		if len(page) < pageSize {
			return backfill, nil
		}
		after = page[len(page)-1].Account
	}
}
