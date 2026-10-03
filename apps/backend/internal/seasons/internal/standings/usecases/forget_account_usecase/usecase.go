package forget_account_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
)

type Forgetter interface {
	DeleteAccount(ctx context.Context, account standings.AccountID) error
}

type UseCase struct {
	forgetter Forgetter
}

func New(forgetter Forgetter) *UseCase {
	return &UseCase{forgetter: forgetter}
}

func (u *UseCase) Execute(ctx context.Context, account standings.AccountID) error {
	if err := u.forgetter.DeleteAccount(ctx, account); err != nil {
		return fmt.Errorf("failed to forget the account's standings: %w", err)
	}
	return nil
}
