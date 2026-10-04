package log_watch_lead

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/lead/usecases/watch_lead_usecase"
)

func New(inner watch_lead_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

// Logs a change of state, not every tick: a failure is retried each second.
type Logged struct {
	inner   watch_lead_usecase.Executor
	logger  *slog.Logger
	failing bool
}

var _ watch_lead_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) error {
	err := l.inner.Execute(ctx)

	switch {
	case err != nil && ctx.Err() == nil && !l.failing:
		l.failing = true
		l.logger.Warn("failed to follow the lead; trying again", slog.Any("error", err))
	case err == nil && l.failing:
		l.failing = false
		l.logger.Info("following the lead again")
	}

	return err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
