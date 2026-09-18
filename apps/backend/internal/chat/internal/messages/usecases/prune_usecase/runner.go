package prune_usecase

import (
	"context"
	"time"
)

// Runner prunes once at start, then every interval. What a prune did is its decorators' to report.
type Runner struct {
	interval time.Duration
	executor Executor
}

func NewRunner(interval time.Duration, executor Executor) *Runner {
	return &Runner{interval: interval, executor: executor}
}

func (r *Runner) Name() string { return "chat-prune" }

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
