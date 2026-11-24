package delete_rejected_pictures_usecase

import (
	"context"
	"log/slog"
	"time"
)

type Executor interface {
	Execute(ctx context.Context) error
}

func NewRunner(every time.Duration, useCase Executor, logger *slog.Logger) Runner {
	return Runner{every: every, useCase: useCase, logger: logger}
}

// Runner deletes the rejected pictures past their time, every so often.
type Runner struct {
	every   time.Duration
	useCase Executor
	logger  *slog.Logger
}

func (r Runner) Name() string { return "flags-rejected-pictures" }

func (r Runner) Run(ctx context.Context) {
	ticker := time.NewTicker(r.every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.useCase.Execute(ctx); err != nil {
				r.logger.Error("failed to delete the rejected pictures past their time", slog.Any("error", err))
			}
		}
	}
}
