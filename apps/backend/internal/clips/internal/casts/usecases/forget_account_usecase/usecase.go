package forget_account_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
)

type Casts interface {
	ForgetAccount(ctx context.Context, account casts.AccountID) error
}

func New(casts Casts) *UseCase {
	return &UseCase{casts: casts}
}

type UseCase struct {
	casts Casts
}

func (u *UseCase) Execute(ctx context.Context, account casts.AccountID) error {
	if err := u.casts.ForgetAccount(ctx, account); err != nil {
		return fmt.Errorf("failed to forget the account's numbers in the casts: %w", err)
	}
	return nil
}
