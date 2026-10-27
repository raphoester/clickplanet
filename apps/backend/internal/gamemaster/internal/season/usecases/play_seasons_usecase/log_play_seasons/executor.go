package log_play_seasons

import (
	"context"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/usecases/play_seasons_usecase"
)

func New(inner play_seasons_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  play_seasons_usecase.Executor
	logger *slog.Logger
}

var _ play_seasons_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) ([]season.Season, error) {
	told, err := l.inner.Execute(ctx)
	for _, each := range told {
		snapshot := each.TakeSnapshot()
		l.logger.Warn("the season plays", slog.Uint64("season", uint64(snapshot.Number)),
			slog.String("starts_at", snapshot.StartsAt.UTC().Format(time.RFC3339)),
			slog.String("ends_at", snapshot.EndsAt.UTC().Format(time.RFC3339)))
	}
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to play the seasons", slog.Any("error", err))
	}
	return told, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
