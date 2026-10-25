package relay_ledger_usecase

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

func (r *Runner) Name() string { return "planet-ledger-relay" }

func (r *Runner) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		r.relay(ctx)

		select {
		case <-ticker.C:
		case <-ctx.Done():
			r.relay(context.WithoutCancel(ctx))
			return
		}
	}
}

func (r *Runner) relay(ctx context.Context) {
	for {
		told, err := r.executor.Execute(ctx)
		if err != nil || told < Batch {
			return
		}
	}
}
