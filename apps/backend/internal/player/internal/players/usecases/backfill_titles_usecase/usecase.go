package backfill_titles_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Stats interface {
	StatsAfter(ctx context.Context, after players.AccountID, limit int) ([]players.Stats, error)
}

type Accounts interface {
	CreationDates(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]time.Time, error)
}

type Titles interface {
	GrantTitles(ctx context.Context, grants players.Grants, at time.Time) error
	BackfilledTitles(ctx context.Context) (players.TitleIDs, error)
	SaveBackfilledTitles(ctx context.Context, titles players.TitleIDs, at time.Time) error
}

type Executor interface {
	Execute(ctx context.Context) (Backfill, error)
}

type Backfill struct {
	Titles   players.TitleIDs
	Accounts int
}

const pageSize = 500

type UseCase struct {
	stats    Stats
	accounts Accounts
	titles   Titles
	catalog  players.Catalog
	clock    cptime.Clock
}

var _ Executor = (*UseCase)(nil)

func New(stats Stats, accounts Accounts, titles Titles, catalog players.Catalog, clock cptime.Clock) *UseCase {
	return &UseCase{stats: stats, accounts: accounts, titles: titles, catalog: catalog, clock: clock}
}

func (u *UseCase) Execute(ctx context.Context) (Backfill, error) {
	done, err := u.titles.BackfilledTitles(ctx)
	if err != nil {
		return Backfill{}, fmt.Errorf("failed to read the backfilled titles: %w", err)
	}

	pending := u.catalog.Without(done)
	if len(pending) == 0 {
		return Backfill{}, nil
	}

	at := u.clock.Now()
	backfill := Backfill{Titles: pending.IDs()}
	var after players.AccountID
	for {
		page, err := u.stats.StatsAfter(ctx, after, pageSize)
		if err != nil {
			return backfill, fmt.Errorf("failed to read a page of stats: %w", err)
		}

		created, err := u.accounts.CreationDates(ctx, players.AccountsOf(page))
		if err != nil {
			return backfill, fmt.Errorf("failed to ask when a page of accounts was made: %w", err)
		}

		grants := pending.GrantsFor(players.CareersOf(page, created))
		if err := u.titles.GrantTitles(ctx, grants, at); err != nil {
			return backfill, fmt.Errorf("failed to grant the titles: %w", err)
		}
		backfill.Accounts += len(grants)

		if len(page) < pageSize {
			break
		}
		after = page[len(page)-1].Account
	}

	if err := u.titles.SaveBackfilledTitles(ctx, backfill.Titles, at); err != nil {
		return backfill, fmt.Errorf("failed to save the backfilled titles: %w", err)
	}
	return backfill, nil
}
