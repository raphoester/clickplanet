package players

import "time"

type Career struct {
	Stats     Stats
	CreatedAt time.Time
}

func CareersOf(page []Stats, created map[AccountID]time.Time) []Career {
	careers := make([]Career, 0, len(page))
	for _, stats := range page {
		careers = append(careers, Career{Stats: stats, CreatedAt: created[stats.Account]})
	}
	return careers
}

func AccountsOf(page []Stats) []AccountID {
	accounts := make([]AccountID, 0, len(page))
	for _, stats := range page {
		accounts = append(accounts, stats.Account)
	}
	return accounts
}
