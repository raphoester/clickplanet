package play_schedule_usecase

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

func (r *Runner) Name() string { return "seasons-rule-schedule" }

func (r *Runner) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		_, _, _ = r.executor.Execute(ctx)

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}
