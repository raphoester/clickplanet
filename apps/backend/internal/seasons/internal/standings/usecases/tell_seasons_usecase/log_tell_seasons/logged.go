package log_tell_seasons

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/tell_seasons_usecase"
)

func New(inner tell_seasons_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  tell_seasons_usecase.Executor
	logger *slog.Logger
}

var _ tell_seasons_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context, told []calendar.Number) ([]tell_seasons_usecase.Told, error) {
	now, err := l.inner.Execute(ctx, told)

	for _, season := range now {
		l.logger.Info("told each player its season",
			slog.Uint64("season", uint64(season.Season)),
			slog.Int("players", season.Players))
	}
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to tell the players their season", slog.Any("error", err))
	}

	return now, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
