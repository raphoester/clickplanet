package titles

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Career struct {
	Stats   players.Stats
	Account players.Account
}

func CareersOf(page []players.Stats, accounts map[players.AccountID]players.Account) []Career {
	careers := make([]Career, 0, len(page))
	for _, stats := range page {
		careers = append(careers, Career{Stats: stats, Account: accounts[stats.Account]})
	}
	return careers
}

func AccountsOf(page []players.Stats) []players.AccountID {
	accounts := make([]players.AccountID, 0, len(page))
	for _, stats := range page {
		accounts = append(accounts, stats.Account)
	}
	return accounts
}
