package titles

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Career struct {
	Stats     players.Stats
	CreatedAt time.Time
}

func CareersOf(page []players.Stats, created map[players.AccountID]time.Time) []Career {
	careers := make([]Career, 0, len(page))
	for _, stats := range page {
		careers = append(careers, Career{Stats: stats, CreatedAt: created[stats.Account]})
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
