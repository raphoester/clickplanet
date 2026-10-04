package titles

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Career struct {
	stats   players.Stats
	account players.Account
}

func CareerOf(stats players.Stats, account players.Account) Career {
	return Career{stats: stats, account: account}
}

func (c Career) Stats() players.Stats { return c.stats }

func (c Career) Account() players.Account { return c.account }

func CareersOf(page []players.Stats, accounts map[players.AccountID]players.Account) []Career {
	careers := make([]Career, 0, len(page))
	for _, stats := range page {
		careers = append(careers, CareerOf(stats, accounts[stats.Account()]))
	}
	return careers
}

func AccountsOf(page []players.Stats) []players.AccountID {
	accounts := make([]players.AccountID, 0, len(page))
	for _, stats := range page {
		accounts = append(accounts, stats.Account())
	}
	return accounts
}
