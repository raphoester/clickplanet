package prune_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Pruner interface {
	DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error)
	DeleteOldestBeyond(ctx context.Context, kept int) (int64, error)
}

type Pruned struct {
	Expired int64
	Excess  int64
}

type Executor interface {
	Execute(ctx context.Context) (Pruned, error)
}

// timeout bounds one prune; each chunk the store deletes is committed on its own.
const timeout = time.Minute

func New(retention time.Duration, maxEvents int, clock cptime.Clock, pruner Pruner) *UseCase {
	return &UseCase{retention: retention, maxEvents: maxEvents, clock: clock, pruner: pruner}
}

type UseCase struct {
	retention time.Duration
	maxEvents int
	clock     cptime.Clock
	pruner    Pruner
}

var _ Executor = (*UseCase)(nil)

// Execute deletes by age first, so the cap only ever takes rows the retention would have kept.
func (u *UseCase) Execute(ctx context.Context) (Pruned, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var pruned Pruned

	expired, err := u.pruner.DeleteBefore(ctx, u.clock.Now().Add(-u.retention))
	pruned.Expired = expired
	if err != nil {
		return pruned, fmt.Errorf("failed to delete the expired activity: %w", err)
	}

	excess, err := u.pruner.DeleteOldestBeyond(ctx, u.maxEvents)
	pruned.Excess = excess
	if err != nil {
		return pruned, fmt.Errorf("failed to delete the activity past its cap: %w", err)
	}

	return pruned, nil
}
