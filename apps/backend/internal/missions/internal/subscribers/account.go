package subscribers

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
)

// A flag's fact counts without an account: a moderator's write, a fortify.
func AccountOrNone(value string) missions.AccountID {
	account, err := missions.AccountIDOf(value)
	if err != nil {
		return missions.AccountID{}
	}
	return account
}
