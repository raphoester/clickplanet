package prune_usecase

import (
	"context"
	"time"
)

type Runner struct {
	interval  time.Duration
	executors []Executor
}

func NewRunner(interval time.Duration, executors ...Executor) *Runner {
	return &Runner{interval: interval, executors: executors}
}

func (r *Runner) Name() string { return "support-prune" }

func (r *Runner) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		for _, executor := range r.executors {
			_, _ = executor.Execute(ctx)
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}
