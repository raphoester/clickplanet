package log_converge_rules

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/finale/usecases/converge_rules_usecase"
)

func New(inner converge_rules_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

// Logs a change of state, not every tick: a failure is retried each second.
type Logged struct {
	inner   converge_rules_usecase.Executor
	logger  *slog.Logger
	failing bool
}

var _ converge_rules_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) error {
	err := l.inner.Execute(ctx)

	switch {
	case err != nil && ctx.Err() == nil && !l.failing:
		l.failing = true
		l.logger.Warn("failed to set the season's rules on planet; trying again", slog.Any("error", err))
	case err == nil && l.failing:
		l.failing = false
		l.logger.Info("the season's rules are set on planet again")
	}

	return err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
