package log_play_arenas

import (
	"context"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/usecases/play_arenas_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
)

func New(inner play_arenas_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  play_arenas_usecase.Executor
	logger *slog.Logger
}

var _ play_arenas_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context, plan moments.Plan) ([]play_arenas_usecase.Call, error) {
	calls, err := l.inner.Execute(ctx, plan)
	for _, call := range calls {
		attrs := []any{
			slog.String("command", call.Command.Action.String()), slog.String("arena", string(call.Command.Arena)),
			slog.String("at", call.Command.At.UTC().Format(time.RFC3339)),
		}
		if call.Refusal != nil {
			l.logger.Error("planet refused an arena command", append(attrs, slog.Any("error", call.Refusal))...)
			continue
		}
		l.logger.Warn("planet answered", attrs...)
	}
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to play the arena battles", slog.Any("error", err))
	}
	return calls, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
