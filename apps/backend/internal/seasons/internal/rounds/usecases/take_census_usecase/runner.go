package take_census_usecase

import (
	"context"
	"time"
)

type Runner struct {
	interval time.Duration
	executor Executor
}

func NewRunner(config Config, executor Executor) *Runner {
	return &Runner{interval: config.WithDefaults().Interval, executor: executor}
}

func (r *Runner) Name() string { return "seasons-census" }

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
