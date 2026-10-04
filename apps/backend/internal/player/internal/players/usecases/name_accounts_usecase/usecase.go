package name_accounts_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Stats interface {
	StatsAfter(ctx context.Context, after players.AccountID, limit int) ([]players.Stats, error)
	Names(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]players.Name, error)
}

type Accounts interface {
	Accounts(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]players.Account, error)
}

type Names interface {
	Assign(ctx context.Context, account players.AccountID, at time.Time) error
}

type Executor interface {
	Execute(ctx context.Context) (int, error)
}

const pageSize = 500

type UseCase struct {
	stats    Stats
	accounts Accounts
	names    Names
	clock    cptime.Clock
}

var _ Executor = (*UseCase)(nil)

func New(stats Stats, accounts Accounts, names Names, clock cptime.Clock) *UseCase {
	return &UseCase{stats: stats, accounts: accounts, names: names, clock: clock}
}

func (u *UseCase) Execute(ctx context.Context) (int, error) {
	at := u.clock.Now()
	var (
		named int
		after players.AccountID
	)
	for {
		page, err := u.stats.StatsAfter(ctx, after, pageSize)
		if err != nil {
			return named, fmt.Errorf("failed to read a page of stats: %w", err)
		}
		accounts := players.AccountsOf(page)
		known, err := u.accounts.Accounts(ctx, accounts)
		if err != nil {
			return named, fmt.Errorf("failed to ask auth about a page of accounts: %w", err)
		}
		names, err := u.stats.Names(ctx, accounts)
		if err != nil {
			return named, fmt.Errorf("failed to read the names of a page: %w", err)
		}
		for _, account := range players.NamelessLinked(page, known, names) {
			if err := u.names.Assign(ctx, account, at); err != nil {
				return named, fmt.Errorf("failed to name an account: %w", err)
			}
			named++
		}
		if len(page) < pageSize {
			return named, nil
		}
		after = page[len(page)-1].Account
	}
}
