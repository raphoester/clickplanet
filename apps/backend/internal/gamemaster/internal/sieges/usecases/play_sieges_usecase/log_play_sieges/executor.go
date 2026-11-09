package log_play_sieges

import (
	"context"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/sieges/usecases/play_sieges_usecase"
)

func New(inner play_sieges_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  play_sieges_usecase.Executor
	logger *slog.Logger
}

var _ play_sieges_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) (play_sieges_usecase.Tick, error) {
	tick, err := l.inner.Execute(ctx)
	for _, judgement := range tick.Judged {
		l.logger.Warn("a battle is judged", slog.String("round", string(judgement.Round)),
			slog.String("mode", judgement.Mode.String()))
	}
	for _, command := range tick.Sent {
		l.logger.Warn("planet is commanded", slog.String("command", command.Action.String()),
			slog.String("arena", string(command.Arena)), slog.String("at", command.At.UTC().Format(time.RFC3339)))
	}
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to play the sieges", slog.Any("error", err))
	}
	return tick, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
