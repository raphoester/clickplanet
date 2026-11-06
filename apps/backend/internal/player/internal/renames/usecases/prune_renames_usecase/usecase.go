package prune_renames_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Pruner interface {
	DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

type Executor interface {
	Execute(ctx context.Context) (int64, error)
}

const timeout = 30 * time.Second

func New(retention time.Duration, clock cptime.Clock, renames Pruner) *UseCase {
	return &UseCase{retention: retention, clock: clock, renames: renames}
}

type UseCase struct {
	retention time.Duration
	clock     cptime.Clock
	renames   Pruner
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	deleted, err := u.renames.DeleteBefore(ctx, u.clock.Now().Add(-u.retention))
	if err != nil {
		return deleted, fmt.Errorf("failed to prune the renames: %w", err)
	}
	return deleted, nil
}
