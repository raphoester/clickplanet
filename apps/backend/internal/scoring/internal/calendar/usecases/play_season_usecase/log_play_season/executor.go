package log_play_season

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar/usecases/play_season_usecase"
)

func New(inner play_season_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  play_season_usecase.Executor
	logger *slog.Logger
}

var _ play_season_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context, decision calendar.Decision) (bool, error) {
	took, err := l.inner.Execute(ctx, decision)
	attrs := []any{
		slog.Uint64("season", uint64(decision.Window.Number)),
		slog.Time("startsAt", decision.Window.StartsAt), slog.Time("endsAt", decision.Window.EndsAt),
	}
	if took {
		l.logger.Warn("season in force", attrs...)
	}
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to play the season", append(attrs, slog.Any("error", err))...)
	}
	return took, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
