package titles

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/finales"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/podiums"
)

type Career struct {
	stats   players.Stats
	account players.Account
	fights  finales.Fights
	places  podiums.Places
}

func CareerOf(stats players.Stats, account players.Account, fights finales.Fights, places podiums.Places) Career {
	return Career{stats: stats, account: account, fights: fights, places: places}
}

func (c Career) Stats() players.Stats { return c.stats }

func (c Career) Account() players.Account { return c.account }

func CareersOf(
	page []players.Stats, accounts map[players.AccountID]players.Account, fights map[players.AccountID]finales.Fights,
	places map[players.AccountID]podiums.Places,
) []Career {
	careers := make([]Career, 0, len(page))
	for _, stats := range page {
		account := stats.Account()
		careers = append(careers, CareerOf(stats, accounts[account], fights[account], places[account]))
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

type Podiums interface {
	Places(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]podiums.Places, error)
}

// Careers puts together what auth says of an account, the finales it fought and its podium places, beside its stats.
type Careers struct {
	accounts Accounts
	fights   Fights
	podiums  Podiums
}

func NewCareers(accounts Accounts, fights Fights, podiums Podiums) Careers {
	return Careers{accounts: accounts, fights: fights, podiums: podiums}
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
	places, err := c.podiums.Places(ctx, []players.AccountID{stats.Account()})
	if err != nil {
		return Career{}, fmt.Errorf("failed to read the account's podium places: %w", err)
	}
	return CareerOf(stats, account, fights[stats.Account()], places[stats.Account()]), nil
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
	places, err := c.podiums.Places(ctx, accounts)
	if err != nil {
		return nil, fmt.Errorf("failed to read the podium places of a page of accounts: %w", err)
	}
	return CareersOf(page, known, fights, places), nil
}
