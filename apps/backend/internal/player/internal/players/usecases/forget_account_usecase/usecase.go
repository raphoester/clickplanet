// Package forget_account_usecase deletes what this module keeps for an account that is gone.
package forget_account_usecase

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Accounts interface {
	DeleteAccount(account players.AccountID)
}

type UseCase struct {
	accounts Accounts
}

func New(accounts Accounts) *UseCase {
	return &UseCase{accounts: accounts}
}

func (u *UseCase) Execute(account players.AccountID) {
	u.accounts.DeleteAccount(account)
}
