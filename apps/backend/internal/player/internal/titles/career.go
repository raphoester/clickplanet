package titles

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/seasons"
)

type Career struct {
	stats   players.Stats
	account players.Account
	seasons seasons.Record
}

func CareerOf(stats players.Stats, account players.Account, played seasons.Record) Career {
	return Career{stats: stats, account: account, seasons: played}
}

func (c Career) Stats() players.Stats { return c.stats }

func (c Career) Account() players.Account { return c.account }

func CareersOf(page []players.Stats, accounts map[players.AccountID]players.Account, played map[players.AccountID]seasons.Record) []Career {
	careers := make([]Career, 0, len(page))
	for _, stats := range page {
		careers = append(careers, CareerOf(stats, accounts[stats.Account()], played[stats.Account()]))
	}
	return careers
}

type Accounts interface {
	Account(ctx context.Context, account players.AccountID) (players.Account, error)
	Accounts(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]players.Account, error)
}

type Records interface {
	Record(ctx context.Context, account players.AccountID) (seasons.Record, error)
	Records(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]seasons.Record, error)
}

type Careers struct {
	accounts Accounts
	records  Records
}

func NewCareers(accounts Accounts, records Records) Careers {
	return Careers{accounts: accounts, records: records}
}

func (c Careers) Of(ctx context.Context, stats players.Stats) (Career, error) {
	known, err := c.accounts.Account(ctx, stats.Account())
	if err != nil {
		return Career{}, fmt.Errorf("failed to ask auth about the account: %w", err)
	}
	played, err := c.records.Record(ctx, stats.Account())
	if err != nil {
		return Career{}, fmt.Errorf("failed to read the seasons played: %w", err)
	}
	return CareerOf(stats, known, played), nil
}

func (c Careers) Page(ctx context.Context, page []players.Stats) ([]Career, error) {
	accounts := players.AccountsOf(page)
	known, err := c.accounts.Accounts(ctx, accounts)
	if err != nil {
		return nil, fmt.Errorf("failed to ask auth about a page of accounts: %w", err)
	}
	played, err := c.records.Records(ctx, accounts)
	if err != nil {
		return nil, fmt.Errorf("failed to read the seasons a page played: %w", err)
	}
	return CareersOf(page, known, played), nil
}
