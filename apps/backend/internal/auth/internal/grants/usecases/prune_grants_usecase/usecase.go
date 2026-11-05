package prune_grants_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Pruner interface {
	PruneGrants(ctx context.Context, expiredBy time.Time, limit int) (int, error)
}

type Executor interface {
	Execute(ctx context.Context) (int, error)
}

const (
	batch        = 1000
	batchTimeout = 30 * time.Second
)

type UseCase struct {
	grants Pruner
	clock  cptime.Clock
}

var _ Executor = (*UseCase)(nil)

func New(grants Pruner, clock cptime.Clock) *UseCase {
	return &UseCase{grants: grants, clock: clock}
}

// Deletes every grant past its lifetime. Nothing else keeps one, so nothing is told.
func (u *UseCase) Execute(ctx context.Context) (int, error) {
	expiredBy := u.clock.Now()

	total := 0
	for {
		pruned, err := u.pruneBatch(ctx, expiredBy)
		total += pruned
		if err != nil {
			return total, err
		}
		if pruned < batch {
			return total, nil
		}
	}
}

func (u *UseCase) pruneBatch(ctx context.Context, expiredBy time.Time) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, batchTimeout)
	defer cancel()

	pruned, err := u.grants.PruneGrants(ctx, expiredBy, batch)
	if err != nil {
		return 0, fmt.Errorf("failed to prune a batch: %w", err)
	}
	return pruned, nil
}
