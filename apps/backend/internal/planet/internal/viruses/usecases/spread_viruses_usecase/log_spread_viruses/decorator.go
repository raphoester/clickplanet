package log_spread_viruses

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/viruses/usecases/spread_viruses_usecase"
)

func New(inner spread_viruses_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  spread_viruses_usecase.Executor
	logger *slog.Logger
}

var _ spread_viruses_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) (int, error) {
	spread, err := l.inner.Execute(ctx)
	if err != nil {
		l.logger.Error("failed to spread the viruses due, retrying next tick", slog.Any("error", err))
	}
	return spread, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
