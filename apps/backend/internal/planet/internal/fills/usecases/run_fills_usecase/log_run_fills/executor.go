package log_run_fills

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills/usecases/run_fills_usecase"
)

func New(inner run_fills_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  run_fills_usecase.Executor
	logger *slog.Logger
}

var _ run_fills_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) ([]fills.Fill, error) {
	finished, err := l.inner.Execute(ctx)
	for _, fill := range finished {
		kept := fill.TakeSnapshot()
		l.logger.Warn("charges filled", slog.String("request", string(kept.ID)), slog.Int("accounts", kept.Filled),
			slog.Duration("took", kept.FinishedAt.Sub(kept.StartedAt)))
	}
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to fill the charges", slog.Any("error", err))
	}
	return finished, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
