package detonate_mines_usecase

import (
	"context"
	"time"
)

type Runner struct {
	fuse     <-chan struct{}
	interval time.Duration
	executor Executor
}

// NewRunner sets the mines off as soon as the fuse says a take tripped one, and each interval all the same, for a
// failure the fuse no longer says.
func NewRunner(fuse <-chan struct{}, interval time.Duration, executor Executor) *Runner {
	return &Runner{fuse: fuse, interval: interval, executor: executor}
}

func (r *Runner) Name() string { return "planet-mines" }

func (r *Runner) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-r.fuse:
			_, _ = r.executor.Execute(ctx)
		case <-ticker.C:
			_, _ = r.executor.Execute(ctx)
		case <-ctx.Done():
			return
		}
	}
}
