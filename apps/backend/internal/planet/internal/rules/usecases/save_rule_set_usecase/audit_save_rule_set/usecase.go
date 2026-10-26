package audit_save_rule_set

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/rule_set_message"
)

type UseCase interface {
	Execute(ctx context.Context, in rules.Snapshot) error
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

func (a *Audited) Execute(ctx context.Context, in rules.Snapshot) error {
	err := a.inner.Execute(ctx, in)

	document, _ := rule_set_message.Document(in)
	attrs := []any{slog.String("set", string(in.Name)), slog.String("rules", string(document))}

	if err != nil {
		a.logger.Warn("admin rule set refused", append(attrs, slog.Any("error", err))...)
		return err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin rule set saved", attrs...)

	return nil
}
