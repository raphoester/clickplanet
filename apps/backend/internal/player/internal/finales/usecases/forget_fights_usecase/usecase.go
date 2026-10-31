package forget_fights_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Fights interface {
	DeleteAccount(ctx context.Context, account players.AccountID) error
}

type UseCase struct {
	fights Fights
}

func New(fights Fights) *UseCase {
	return &UseCase{fights: fights}
}

func (u *UseCase) Execute(ctx context.Context, account players.AccountID) error {
	if err := u.fights.DeleteAccount(ctx, account); err != nil {
		return fmt.Errorf("failed to forget the account's fights: %w", err)
	}
	return nil
}
