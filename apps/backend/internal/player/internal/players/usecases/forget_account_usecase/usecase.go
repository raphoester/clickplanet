// Package forget_account_usecase deletes what this module keeps for an account that is gone.
package forget_account_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Accounts interface {
	DeleteAccount(ctx context.Context, account players.AccountID) error
}

type UseCase struct {
	accounts Accounts
}

func New(accounts Accounts) *UseCase {
	return &UseCase{accounts: accounts}
}

func (u *UseCase) Execute(ctx context.Context, account players.AccountID) error {
	if err := u.accounts.DeleteAccount(ctx, account); err != nil {
		return fmt.Errorf("failed to forget the account: %w", err)
	}
	return nil
}
