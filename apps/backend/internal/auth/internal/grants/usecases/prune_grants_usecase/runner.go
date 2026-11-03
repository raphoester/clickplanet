package prune_grants_usecase

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/grants"
)

type Runner struct {
	interval time.Duration
	executor Executor
}

// Once per grants.Lifetime: no grant outlives it by more than as much again.
func NewRunner(executor Executor) *Runner {
	return &Runner{interval: grants.Lifetime, executor: executor}
}

func (r *Runner) Name() string { return "auth-grant-prune" }

func (r *Runner) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		_, _ = r.executor.Execute(ctx)

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}
