package forget_account_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/optouts"
)

type OptOuts interface {
	Forget(ctx context.Context, account optouts.AccountID) error
}

func New(optOuts OptOuts) *UseCase {
	return &UseCase{optOuts: optOuts}
}

type UseCase struct {
	optOuts OptOuts
}

func (u *UseCase) Execute(ctx context.Context, account optouts.AccountID) error {
	if err := u.optOuts.Forget(ctx, account); err != nil {
		return fmt.Errorf("failed to forget the account's opt-out: %w", err)
	}
	return nil
}
