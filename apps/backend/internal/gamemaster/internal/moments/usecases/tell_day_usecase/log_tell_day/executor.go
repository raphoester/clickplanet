package log_tell_day

import (
	"context"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/tell_day_usecase"
)

func New(inner tell_day_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  tell_day_usecase.Executor
	logger *slog.Logger
}

var _ tell_day_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) ([]moments.DayStarted, error) {
	told, err := l.inner.Execute(ctx)
	for _, started := range told {
		l.logger.Info("day told", slog.String("round", string(moments.DayRoundOf(started.Day))),
			slog.String("starts_at", started.Day.StartsAt.UTC().Format(time.RFC3339)),
			slog.String("ends_at", started.Day.EndsAt.UTC().Format(time.RFC3339)))
	}
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to tell the day", slog.Any("error", err))
	}
	return told, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
