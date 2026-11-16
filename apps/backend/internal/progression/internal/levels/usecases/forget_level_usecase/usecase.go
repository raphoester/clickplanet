package forget_level_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
)

type Levels interface {
	Forget(ctx context.Context, accounts ...accounts.AccountID) error
}

type UseCase struct {
	levels Levels
}

func New(levels Levels) *UseCase {
	return &UseCase{levels: levels}
}

func (u *UseCase) Execute(ctx context.Context, account accounts.AccountID) error {
	if err := u.levels.Forget(ctx, account); err != nil {
		return fmt.Errorf("failed to forget the level: %w", err)
	}
	return nil
}
