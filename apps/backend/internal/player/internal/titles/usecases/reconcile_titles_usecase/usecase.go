package reconcile_titles_usecase

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
	Accounts(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]players.Account, error)
}

type Titles interface {
	Holdings(ctx context.Context, accounts []players.AccountID) (titles.Holdings, error)
	Grant(ctx context.Context, grants titles.Holdings, at time.Time) error
	Revoke(ctx context.Context, revocations titles.Holdings) error
}

type Executor interface {
	Execute(ctx context.Context) (Reconciled, error)
}

type Reconciled struct {
	Granted int
	Revoked int
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

func (u *UseCase) Execute(ctx context.Context) (Reconciled, error) {
	at := u.clock.Now()
	var (
		reconciled Reconciled
		after      players.AccountID
	)
	for {
		page, err := u.stats.StatsAfter(ctx, after, pageSize)
		if err != nil {
			return reconciled, fmt.Errorf("failed to read a page of stats: %w", err)
		}

		accounts := titles.AccountsOf(page)
		known, err := u.accounts.Accounts(ctx, accounts)
		if err != nil {
			return reconciled, fmt.Errorf("failed to ask auth about a page of accounts: %w", err)
		}
		held, err := u.titles.Holdings(ctx, accounts)
		if err != nil {
			return reconciled, fmt.Errorf("failed to read the titles of a page: %w", err)
		}

		reconciliation := u.catalog.ReconciliationOf(titles.CareersOf(page, known), held)
		if err := u.titles.Grant(ctx, reconciliation.Grants(), at); err != nil {
			return reconciled, fmt.Errorf("failed to grant the titles: %w", err)
		}
		reconciled.Granted += reconciliation.Grants().Len()
		if err := u.titles.Revoke(ctx, reconciliation.Revocations()); err != nil {
			return reconciled, fmt.Errorf("failed to revoke the titles: %w", err)
		}
		reconciled.Revoked += reconciliation.Revocations().Len()

		if len(page) < pageSize {
			return reconciled, nil
		}
		after = page[len(page)-1].Account()
	}
}
