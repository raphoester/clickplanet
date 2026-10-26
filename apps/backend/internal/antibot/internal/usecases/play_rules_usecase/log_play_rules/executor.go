package log_play_rules

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules/rulesmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/play_rules_usecase"
)

func New(inner play_rules_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  play_rules_usecase.Executor
	logger *slog.Logger
}

var _ play_rules_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) (play_rules_usecase.Taken, bool, error) {
	taken, took, err := l.inner.Execute(ctx)
	if took {
		written, _ := rulesmessage.Document.Marshal(rulesmessage.Of(taken.Set))
		l.logger.Warn("antibot rules in force",
			slog.String("set", string(taken.Set.Name)),
			slog.String("asked", string(taken.Asked)),
			slog.String("rules", string(written)))
	}
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to play the antibot rules", slog.Any("error", err))
	}
	return taken, took, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
