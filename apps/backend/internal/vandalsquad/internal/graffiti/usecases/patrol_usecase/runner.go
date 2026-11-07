package patrol_usecase

import (
	"context"
	"log/slog"
	"time"
)

type Executor interface {
	Execute(ctx context.Context) error
}

func NewRunner(interval time.Duration, patrol Executor, logger *slog.Logger) *Runner {
	return &Runner{interval: interval, patrol: patrol, logger: logger}
}

// A patrol every interval, the first one at once.
type Runner struct {
	interval time.Duration
	patrol   Executor
	logger   *slog.Logger
}

func (r *Runner) Name() string { return "vandalsquad-patrol" }

func (r *Runner) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		if err := r.patrol.Execute(ctx); err != nil && ctx.Err() == nil {
			r.logger.ErrorContext(ctx, "the vandal squad's patrol failed", slog.Any("error", err))
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}
