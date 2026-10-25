package audit_apply_rule_set

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/apply_rule_set_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in apply_rule_set_usecase.In) error
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

func (a *Audited) Execute(ctx context.Context, in apply_rule_set_usecase.In) error {
	err := a.inner.Execute(ctx, in)

	attrs := []any{
		slog.String("request", string(in.Request)),
		slog.String("set", string(in.Set)),
		slog.String("source", string(in.Source)),
	}

	if err != nil {
		a.logger.Warn("rule set refused", append(attrs, slog.Any("error", err))...)
		return err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("rule set asked", attrs...)

	return nil
}
