package log_play_resets

import (
	"context"
	"errors"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
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

func (l *Logged) Execute(ctx context.Context, plan moments.Plan) (season.Window, bool, error) {
	starts, done, err := l.inner.Execute(ctx, plan)
	attrs := []any{slog.Uint64("season", uint64(starts.Number)), slog.Time("starts_at", starts.StartsAt)}
	switch {
	case done:
		l.logger.Warn("planet reset the map", attrs...)
	case errors.Is(err, moments.ErrRefused):
		l.logger.Error("planet refused to reset the map", append(attrs, slog.Any("error", err))...)
	case err != nil && ctx.Err() == nil:
		l.logger.Error("failed to ask planet to reset the map", append(attrs, slog.Any("error", err))...)
	}
	return starts, done, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
