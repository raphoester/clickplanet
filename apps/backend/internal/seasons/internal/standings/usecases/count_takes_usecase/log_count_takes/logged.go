package log_count_takes

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/count_takes_usecase"
)

func New(inner count_takes_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  count_takes_usecase.Executor
	logger *slog.Logger
}

var _ count_takes_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) (count_takes_usecase.Out, error) {
	out, err := l.inner.Execute(ctx)

	if out.Began {
		l.logger.Info("the standings count the takes from planet's log", slog.Uint64("from", uint64(out.From)))
	}
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to count the takes into the standings", slog.Uint64("from", uint64(out.From)), slog.Any("error", err))
	}

	return out, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
