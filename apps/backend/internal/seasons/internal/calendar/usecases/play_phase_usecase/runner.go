package play_phase_usecase

import (
	"context"
	"time"
)

// TODO(broker): seasons tells planet its phase with a loopback call, sent again every second until planet answers,
// and again after each boot; planet drops a request id it applied already. That loop and that dedupe must move to a
// command stream of planet's on Redis. See apps/backend/docs/rules.md.
type Runner struct {
	interval time.Duration
	executor Executor
}

func NewRunner(interval time.Duration, executor Executor) *Runner {
	return &Runner{interval: interval, executor: executor}
}

func (r *Runner) Name() string { return "seasons-phases" }

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
