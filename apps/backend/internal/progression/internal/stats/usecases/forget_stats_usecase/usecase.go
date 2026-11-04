package forget_stats_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
)

type Stats interface {
	Forget(ctx context.Context, account accounts.AccountID) error
}

type UseCase struct {
	stats Stats
}

func New(stats Stats) *UseCase {
	return &UseCase{stats: stats}
}

func (u *UseCase) Execute(ctx context.Context, account accounts.AccountID) error {
	if err := u.stats.Forget(ctx, account); err != nil {
		return fmt.Errorf("failed to forget the stats: %w", err)
	}
	return nil
}
