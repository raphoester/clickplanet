package log_enforce_rules

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/rule_set_message"
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
	if enforced.Name != "" {
		document, _ := rule_set_message.Document(enforced)
		l.logger.Warn("rules in force", slog.String("set", string(enforced.Name)), slog.String("rules", string(document)))
	}
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to enforce the rules", slog.Any("error", err))
	}
	return enforced, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
