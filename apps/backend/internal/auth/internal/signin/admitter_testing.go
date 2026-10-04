//go:build testing

package signin

import "github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"

func AdmissionOf(account accounts.AccountID, outcome accounts.Outcome, setCookie string) *Admission {
	return &Admission{account: account, outcome: outcome, setCookie: setCookie}
}
