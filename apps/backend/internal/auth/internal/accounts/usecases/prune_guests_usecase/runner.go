package prune_guests_usecase

import (
	"context"
	"time"
)

// Runner prunes once at start, then every interval. What a prune did is its decorators' to report.
type Runner struct {
	interval time.Duration
	executor Executor
}

func NewRunner(config Config, executor Executor) *Runner {
	return &Runner{interval: config.WithDefaults().Interval, executor: executor}
}

func (r *Runner) Name() string { return "auth-guest-prune" }

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
