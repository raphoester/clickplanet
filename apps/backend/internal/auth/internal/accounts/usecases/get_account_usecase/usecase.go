package get_account_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

type UseCase struct {
	accounts accounts.AccountFinder
}

func New(accounts accounts.AccountFinder) *UseCase {
	return &UseCase{accounts: accounts}
}

func (u *UseCase) Execute(ctx context.Context, account accounts.AccountID) (*accounts.Account, error) {
	found, err := u.accounts.Account(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to find the account: %w", err)
	}
	return found, nil
}
