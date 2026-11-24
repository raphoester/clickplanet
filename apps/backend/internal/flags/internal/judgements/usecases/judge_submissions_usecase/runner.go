package judge_submissions_usecase

import (
	"context"
	"log/slog"
	"time"
)

type Executor interface {
	Execute(ctx context.Context) (int, error)
}

// The wait after a failed round doubles up to this: an API down is asked again every few minutes, not every second.
const maxBackOff = 5 * time.Minute

func NewRunner(every time.Duration, useCase Executor, logger *slog.Logger) Runner {
	return Runner{every: every, useCase: useCase, logger: logger}
}

// Runner judges the submissions sent, every few seconds: a buyer on Stripe's page meanwhile finds its flag decided.
type Runner struct {
	every   time.Duration
	useCase Executor
	logger  *slog.Logger
}

func (r Runner) Name() string { return "flags-judge" }

func (r Runner) Run(ctx context.Context) {
	wait := r.every
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		judged, err := r.useCase.Execute(ctx)
		switch {
		case err != nil:
			wait = min(wait*2, maxBackOff)
			r.logger.Error("failed to judge the flags sent", slog.Any("error", err), slog.Duration("next", wait))
		case judged > 0:
			wait = r.every
			r.logger.Info("flags sent judged", slog.Int("judged", judged))
		default:
			wait = r.every
		}
	}
}
