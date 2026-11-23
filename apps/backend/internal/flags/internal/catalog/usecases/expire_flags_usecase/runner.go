package expire_flags_usecase

import (
	"context"
	"log/slog"
	"time"
)

type Executor interface {
	Execute(ctx context.Context) (int, error)
}

func NewRunner(every time.Duration, useCase Executor, logger *slog.Logger) Runner {
	return Runner{every: every, useCase: useCase, logger: logger}
}

// Runner closes the flags whose time ran out, every so often: a flag ends at most that long after its end.
type Runner struct {
	every   time.Duration
	useCase Executor
	logger  *slog.Logger
}

func (r Runner) Name() string { return "flags-expiry" }

func (r Runner) Run(ctx context.Context) {
	ticker := time.NewTicker(r.every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			closed, err := r.useCase.Execute(ctx)
			switch {
			case err != nil:
				r.logger.Error("failed to close the flags whose time ran out", slog.Any("error", err))
			case closed > 0:
				r.logger.Info("flags closed at their end", slog.Int("closed", closed))
			}
		}
	}
}
