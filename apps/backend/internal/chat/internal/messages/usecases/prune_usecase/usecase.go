// Package prune_usecase deletes what the chat keeps past its retention: messages, their reactions, and the
// announcements between them.
package prune_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// Pruner is one table of the chat. A reaction is never older than its message, so what outlives a pruned
// message goes on a later prune.
type Pruner interface {
	DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

// Executor is the prune, as the runner calls it and a decorator wraps it.
type Executor interface {
	Execute(ctx context.Context) (int64, error)
}

const timeout = 30 * time.Second

func New(retention time.Duration, clock cptime.Clock, pruners ...Pruner) *UseCase {
	return &UseCase{retention: retention, clock: clock, pruners: pruners}
}

type UseCase struct {
	retention time.Duration
	clock     cptime.Clock
	pruners   []Pruner
}

var _ Executor = (*UseCase)(nil)

// Execute deletes from each table in turn and says how many rows went. It stops at the first failure.
func (u *UseCase) Execute(ctx context.Context) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cutoff := u.clock.Now().Add(-u.retention)

	var total int64
	for _, pruner := range u.pruners {
		deleted, err := pruner.DeleteBefore(ctx, cutoff)
		total += deleted
		if err != nil {
			return total, fmt.Errorf("failed to prune the chat: %w", err)
		}
	}
	return total, nil
}
