package play_plan_usecase

import (
	"context"
	"time"
)

type Runner struct {
	interval time.Duration
	executor Executor
}

func NewRunner(interval time.Duration, executor Executor) *Runner {
	return &Runner{interval: interval, executor: executor}
}

func (r *Runner) Name() string { return "gamemaster-plan" }

func (r *Runner) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		_ = r.executor.Execute(ctx)
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}
