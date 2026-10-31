package titles

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/finales"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Career struct {
	stats   players.Stats
	account players.Account
	fights  finales.Fights
}

func CareerOf(stats players.Stats, account players.Account, fights finales.Fights) Career {
	return Career{stats: stats, account: account, fights: fights}
}

func (c Career) Stats() players.Stats { return c.stats }

func (c Career) Account() players.Account { return c.account }

func CareersOf(
	page []players.Stats, accounts map[players.AccountID]players.Account, fights map[players.AccountID]finales.Fights,
) []Career {
	careers := make([]Career, 0, len(page))
	for _, stats := range page {
		careers = append(careers, CareerOf(stats, accounts[stats.Account()], fights[stats.Account()]))
	}
	return careers
}

type Accounts interface {
	Account(ctx context.Context, account players.AccountID) (players.Account, error)
	Accounts(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]players.Account, error)
}

type Fights interface {
	Fights(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]finales.Fights, error)
}

// Careers puts together what auth says of an account and the finales it fought, beside its stats.
type Careers struct {
	accounts Accounts
	fights   Fights
}

func NewCareers(accounts Accounts, fights Fights) Careers {
	return Careers{accounts: accounts, fights: fights}
}

func (c Careers) Of(ctx context.Context, stats players.Stats) (Career, error) {
	account, err := c.accounts.Account(ctx, stats.Account())
	if err != nil {
		return Career{}, fmt.Errorf("failed to ask auth about the account: %w", err)
	}
	fights, err := c.fights.Fights(ctx, []players.AccountID{stats.Account()})
	if err != nil {
		return Career{}, fmt.Errorf("failed to read the finales the account fought: %w", err)
	}
	return CareerOf(stats, account, fights[stats.Account()]), nil
}

func (c Careers) PageOf(ctx context.Context, page []players.Stats) ([]Career, error) {
	accounts := players.AccountsOf(page)
	known, err := c.accounts.Accounts(ctx, accounts)
	if err != nil {
		return nil, fmt.Errorf("failed to ask auth about a page of accounts: %w", err)
	}
	fights, err := c.fights.Fights(ctx, accounts)
	if err != nil {
		return nil, fmt.Errorf("failed to read the finales a page of accounts fought: %w", err)
	}
	return CareersOf(page, known, fights), nil
}
