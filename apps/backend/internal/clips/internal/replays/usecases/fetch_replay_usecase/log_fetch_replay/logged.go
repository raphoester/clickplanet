package log_fetch_replay

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_replay_usecase"
)

func New(inner fetch_replay_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  fetch_replay_usecase.Executor
	logger *slog.Logger
}

var _ fetch_replay_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) (replays.Replay, error) {
	replay, err := l.inner.Execute(ctx)
	if err == nil {
		l.logger.Info("saved a replay", slog.Int("events", replay.Events),
			slog.Time("since", replay.Window.Since), slog.Time("until", replay.Window.Until))
	}

	return replay, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
