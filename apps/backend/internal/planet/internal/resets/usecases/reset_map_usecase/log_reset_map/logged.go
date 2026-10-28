package log_reset_map

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/resets"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/resets/usecases/reset_map_usecase"
)

func New(inner reset_map_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  reset_map_usecase.Executor
	logger *slog.Logger
}

var _ reset_map_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context, reset resets.Reset) (bool, error) {
	wiped, err := l.inner.Execute(ctx, reset)
	attrs := []any{slog.String("reset", string(reset.ID)), slog.Time("at", reset.At)}
	switch {
	case err != nil:
		l.logger.Error("failed to reset the map", append(attrs, slog.Any("error", err))...)
	case wiped:
		l.logger.Warn("the map is reset", attrs...)
	default:
		l.logger.Info("the map reset was applied already", attrs...)
	}
	return wiped, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
