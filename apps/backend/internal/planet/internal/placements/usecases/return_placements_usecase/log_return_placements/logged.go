package log_return_placements

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/placements/usecases/return_placements_usecase"
)

func New(inner return_placements_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  return_placements_usecase.Executor
	logger *slog.Logger
}

var _ return_placements_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) (int, error) {
	found, err := l.inner.Execute(ctx)
	switch {
	case err != nil:
		l.logger.Error("failed to give back the shields a stop left on their way; the next boot tries again",
			slog.Int("found", found), slog.Any("error", err))
	case found > 0:
		l.logger.Warn("gave back the shields a stop left on their way", slog.Int("shields", found))
	}
	return found, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
