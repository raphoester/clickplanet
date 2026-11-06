package prune_usecase

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

// The tables kept for one retention: the chat's, or the moderation history's, which outlives it.
type Tables struct {
	retention time.Duration
	pruners   []Pruner
}

func KeptFor(retention time.Duration, pruners ...Pruner) Tables {
	return Tables{retention: retention, pruners: pruners}
}

func New(clock cptime.Clock, tables ...Tables) *UseCase {
	return &UseCase{clock: clock, tables: tables}
}

type UseCase struct {
	clock  cptime.Clock
	tables []Tables
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	now := u.clock.Now()
	var total int64
	for _, tables := range u.tables {
		cutoff := now.Add(-tables.retention)
		for _, pruner := range tables.pruners {
			deleted, err := pruner.DeleteBefore(ctx, cutoff)
			total += deleted
			if err != nil {
				return total, fmt.Errorf("failed to prune the chat: %w", err)
			}
		}
	}
	return total, nil
}
