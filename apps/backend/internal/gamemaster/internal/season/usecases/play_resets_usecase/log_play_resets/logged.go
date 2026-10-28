package log_play_resets

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/usecases/play_resets_usecase"
)

func New(inner play_resets_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  play_resets_usecase.Executor
	logger *slog.Logger
}

var _ play_resets_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) (season.Window, bool, error) {
	starts, told, err := l.inner.Execute(ctx)
	if told {
		l.logger.Warn("planet is told to reset the map", slog.Uint64("season", uint64(starts.Number)),
			slog.Time("starts_at", starts.StartsAt))
	}
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to tell planet to reset the map", slog.Any("error", err))
	}
	return starts, told, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
