package log_fetch_history

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_history_usecase"
)

func New(inner fetch_history_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  fetch_history_usecase.Executor
	logger *slog.Logger
}

var _ fetch_history_usecase.Executor = (*Logged)(nil)

// A map left out is a warning, not a failure: the run still has its replay, and tells no saga.
func (l *Logged) Execute(ctx context.Context, turn replays.Turn, until time.Time) (replays.History, error) {
	history, err := l.inner.Execute(ctx, turn, until)
	switch {
	case errors.Is(err, replays.ErrNoSeason):
		l.logger.Info("left the season's map out: no season runs, so no saga is told")
	case err != nil:
		l.logger.Warn("left the season's map out: no saga is told", slog.Any("error", err))
	default:
		l.logger.Info("saved the season's map", slog.Time("since", history.Window.Since), slog.Time("until", history.Window.Until),
			slog.Int("frames", history.Frames), slog.Int("bytes", len(history.Document)))
	}

	return history, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
