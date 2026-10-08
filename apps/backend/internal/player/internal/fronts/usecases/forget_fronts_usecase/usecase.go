package forget_fronts_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Fronts interface {
	DeleteAccount(ctx context.Context, account players.AccountID) error
}

type UseCase struct {
	fronts Fronts
}

func New(fronts Fronts) *UseCase {
	return &UseCase{fronts: fronts}
}

func (u *UseCase) Execute(ctx context.Context, account players.AccountID) error {
	if err := u.fronts.DeleteAccount(ctx, account); err != nil {
		return fmt.Errorf("failed to forget the account's fronts: %w", err)
	}
	return nil
}
