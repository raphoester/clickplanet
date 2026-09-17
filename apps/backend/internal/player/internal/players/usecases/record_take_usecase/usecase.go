// Package record_take_usecase counts a tile an account took, for its stats.
package record_take_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Stats interface {
	RecordTake(ctx context.Context, account players.AccountID, at time.Time) error
}

type UseCase struct {
	stats Stats
}

func New(stats Stats) *UseCase {
	return &UseCase{stats: stats}
}

type In struct {
	Account players.AccountID
	At      time.Time
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	if err := u.stats.RecordTake(ctx, in.Account, in.At); err != nil {
		return fmt.Errorf("failed to record the take: %w", err)
	}
	return nil
}
