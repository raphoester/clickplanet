package log_enforce_rules

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/enforce_rules_usecase"
)

func New(inner enforce_rules_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  enforce_rules_usecase.Executor
	logger *slog.Logger
}

var _ enforce_rules_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) (rules.Snapshot, error) {
	enforced, err := l.inner.Execute(ctx)
	if enforced != (rules.Snapshot{}) {
		l.logger.Warn("rules in force",
			slog.String("set", string(enforced.Name)),
			slog.Float64("tempo", enforced.Tempo),
			slog.Duration("boxMinInterval", enforced.Boxes.MinInterval),
			slog.Duration("boxMaxInterval", enforced.Boxes.MaxInterval),
			slog.Int("boxMaxChargesPerHour", enforced.Boxes.MaxChargesPerHour))
	}
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to enforce the rules", slog.Any("error", err))
	}
	return enforced, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
