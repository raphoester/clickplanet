package log_fetch_cast

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_cast_usecase"
)

func New(inner fetch_cast_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  fetch_cast_usecase.Executor
	logger *slog.Logger
}

var _ fetch_cast_usecase.Executor = (*Logged)(nil)

// A cast left out is a warning, not a failure: the run's clips name nobody. Counts, never a name.
func (l *Logged) Execute(ctx context.Context, window replays.Window) (replays.Cast, error) {
	cast, err := l.inner.Execute(ctx, window)
	if err != nil {
		l.logger.Warn("left the cast out: the clips name nobody", slog.Any("error", err))
		return cast, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}
	l.logger.Info("saved the cast", slog.Int("fighters", cast.Fighters), slog.Int("takes", cast.Takes))
	return cast, nil
}
