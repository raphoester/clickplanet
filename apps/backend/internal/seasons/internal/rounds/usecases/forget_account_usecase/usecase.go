package forget_account_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
)

type Forgetter interface {
	ForgetAccount(ctx context.Context, account rounds.AccountID) error
}

func New(forgetter Forgetter) *UseCase {
	return &UseCase{forgetter: forgetter}
}

// UseCase forgets which tiles a deleted account took in the rounds of their own; its flag keeps them.
type UseCase struct {
	forgetter Forgetter
}

func (u *UseCase) Execute(ctx context.Context, account rounds.AccountID) error {
	if err := u.forgetter.ForgetAccount(ctx, account); err != nil {
		return fmt.Errorf("failed to forget the account's takes in the rounds: %w", err)
	}
	return nil
}
