package scoringread

// Ranked holds for the account in the column a board may rank: a player (players), not asked deleted (departures).
func Ranked(account string) string {
	return `(EXISTS (SELECT 1 FROM players WHERE players.account_id = ` + account + `)
		AND NOT EXISTS (SELECT 1 FROM departures WHERE departures.account_id = ` + account + ` AND departures.leaving))`
}
