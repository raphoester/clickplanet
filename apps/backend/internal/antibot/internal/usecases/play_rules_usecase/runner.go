package play_rules_usecase

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

func (r *Runner) Name() string { return "antibot-rules" }

func (r *Runner) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_, _, _ = r.executor.Execute(ctx)
		case <-ctx.Done():
			return
		}
	}
}
