package get_accounts_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

type Accounts interface {
	Accounts(ctx context.Context, accounts []accounts.AccountID) ([]*accounts.Account, error)
}

type UseCase struct {
	accounts Accounts
}

func New(accounts Accounts) *UseCase {
	return &UseCase{accounts: accounts}
}

func (u *UseCase) Execute(ctx context.Context, asked []accounts.AccountID) ([]*accounts.Account, error) {
	found, err := u.accounts.Accounts(ctx, asked)
	if err != nil {
		return nil, fmt.Errorf("failed to read the accounts: %w", err)
	}
	return found, nil
}
