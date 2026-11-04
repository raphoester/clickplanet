package forget_fights_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
)

type Fights interface {
	DeleteAccount(ctx context.Context, account accounts.AccountID) error
}

type UseCase struct {
	fights Fights
}

func New(fights Fights) *UseCase {
	return &UseCase{fights: fights}
}

func (u *UseCase) Execute(ctx context.Context, account accounts.AccountID) error {
	if err := u.fights.DeleteAccount(ctx, account); err != nil {
		return fmt.Errorf("failed to forget the account's fights: %w", err)
	}
	return nil
}
