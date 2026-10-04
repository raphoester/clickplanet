package forget_baseline_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Baseline interface {
	DeleteAccount(ctx context.Context, account players.AccountID) error
}

type UseCase struct {
	baseline Baseline
}

func New(baseline Baseline) *UseCase {
	return &UseCase{baseline: baseline}
}

func (u *UseCase) Execute(ctx context.Context, account players.AccountID) error {
	if err := u.baseline.DeleteAccount(ctx, account); err != nil {
		return fmt.Errorf("failed to forget the account's baseline: %w", err)
	}
	return nil
}
