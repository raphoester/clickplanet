package run_fills_usecase

import (
	"context"
	"time"
)

func NewRunner(executor Executor, retry time.Duration) *Runner {
	return &Runner{executor: executor, retry: retry, wake: make(chan struct{}, 1)}
}

// Runner runs the fills at boot, so one cut short by a restart goes on, then each time one is started.
type Runner struct {
	executor Executor
	retry    time.Duration
	wake     chan struct{}
}

func (r *Runner) Name() string { return "planet-charge-fills" }

func (r *Runner) Wake() {
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

func (r *Runner) Run(ctx context.Context) {
	for {
		_, err := r.executor.Execute(ctx)

		select {
		case <-r.wake:
		case <-r.retryAfter(err):
		case <-ctx.Done():
			return
		}
	}
}

// retryAfter fires once the retry has passed after a failure, and never after a run that went through.
func (r *Runner) retryAfter(err error) <-chan time.Time {
	if err == nil {
		return nil
	}
	return time.After(r.retry)
}
