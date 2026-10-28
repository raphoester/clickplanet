package forget_account_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
)

type Appearances interface {
	ForgetAccount(ctx context.Context, account casts.AccountID) error
}

func New(appearances Appearances) *UseCase {
	return &UseCase{appearances: appearances}
}

type UseCase struct {
	appearances Appearances
}

func (u *UseCase) Execute(ctx context.Context, account casts.AccountID) error {
	if err := u.appearances.ForgetAccount(ctx, account); err != nil {
		return fmt.Errorf("failed to forget the posts that named the account: %w", err)
	}
	return nil
}
