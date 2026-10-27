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
}

func NewRunner(executor Executor, clock cptime.Clock, retry time.Duration) *Runner {
	return &Runner{executor: executor, clock: clock, retry: retry}
}

func (r *Runner) Name() string { return "seasons-rounds" }

func (r *Runner) Run(ctx context.Context) {
	for {
		report, err := r.executor.Execute(ctx)
		wait := r.retry
		if err == nil {
			if report.Next.IsZero() {
				<-ctx.Done()
				return
			}
			wait = report.Next.Sub(r.clock.Now())
		}

		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return
		}
	}
}
