package titles

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/finales"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
)

type Career struct {
	stats   stats.Stats
	account accounts.Account
	fights  finales.Fights
}

func CareerOf(stats stats.Stats, account accounts.Account, fights finales.Fights) Career {
	return Career{stats: stats, account: account, fights: fights}
}

func (c Career) Stats() stats.Stats { return c.stats }

func (c Career) Account() accounts.Account { return c.account }

func CareersOf(
	page []stats.Stats, accounts map[accounts.AccountID]accounts.Account, fights map[accounts.AccountID]finales.Fights,
) []Career {
	careers := make([]Career, 0, len(page))
	for _, stats := range page {
		careers = append(careers, CareerOf(stats, accounts[stats.Account()], fights[stats.Account()]))
	}
	return careers
}

type Accounts interface {
	Account(ctx context.Context, account accounts.AccountID) (accounts.Account, error)
	Accounts(ctx context.Context, accounts []accounts.AccountID) (map[accounts.AccountID]accounts.Account, error)
}

type Fights interface {
	Fights(ctx context.Context, accounts []accounts.AccountID) (map[accounts.AccountID]finales.Fights, error)
}

// Careers puts together what player says of an account and the finales it fought, beside its stats.
type Careers struct {
	accounts Accounts
	fights   Fights
}

func NewCareers(accounts Accounts, fights Fights) Careers {
	return Careers{accounts: accounts, fights: fights}
}

func (c Careers) Of(ctx context.Context, stats stats.Stats) (Career, error) {
	account, err := c.accounts.Account(ctx, stats.Account())
	if err != nil {
		return Career{}, fmt.Errorf("failed to ask player about the account: %w", err)
	}
	fights, err := c.fights.Fights(ctx, []accounts.AccountID{stats.Account()})
	if err != nil {
		return Career{}, fmt.Errorf("failed to read the finales the account fought: %w", err)
	}
	return CareerOf(stats, account, fights[stats.Account()]), nil
}

func (c Careers) PageOf(ctx context.Context, page []stats.Stats) ([]Career, error) {
	accounts := stats.AccountsOf(page)
	known, err := c.accounts.Accounts(ctx, accounts)
	if err != nil {
		return nil, fmt.Errorf("failed to ask player about a page of accounts: %w", err)
	}
	fights, err := c.fights.Fights(ctx, accounts)
	if err != nil {
		return nil, fmt.Errorf("failed to read the finales a page of accounts fought: %w", err)
	}
	return CareersOf(page, known, fights), nil
}
