package close_ended_rounds_usecase

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const Retry = 5 * time.Second

type Runner struct {
	executor Executor
	clock    cptime.Clock
	retry    time.Duration
	wake     chan struct{}
}

func NewRunner(executor Executor, clock cptime.Clock, retry time.Duration) *Runner {
	return &Runner{executor: executor, clock: clock, retry: retry, wake: make(chan struct{}, 1)}
}

func (r *Runner) Name() string { return "seasons-rounds" }

// Wake runs the executor again at once: a round of its own just opened may end before the one the runner waits for.
func (r *Runner) Wake() {
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

func (r *Runner) Run(ctx context.Context) {
	for {
		report, err := r.executor.Execute(ctx)
		var due <-chan time.Time
		switch {
		case err != nil:
			due = time.After(r.retry)
		case !report.Next.IsZero():
			due = time.After(report.Next.Sub(r.clock.Now()))
		}

		select {
		case <-due:
		case <-r.wake:
		case <-ctx.Done():
			return
		}
	}
}
