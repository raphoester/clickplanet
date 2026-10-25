package audit_override_rules

import (
	"context"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/override_rules_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in override_rules_usecase.In) error
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

func (a *Audited) Execute(ctx context.Context, in override_rules_usecase.In) error {
	err := a.inner.Execute(ctx, in)

	attrs := []any{slog.String("set", string(in.Set)), slog.String("until", in.Until.UTC().Format(time.RFC3339))}

	if err != nil {
		a.logger.Warn("admin rule override refused", append(attrs, slog.Any("error", err))...)
		return err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin rule override", attrs...)

	return nil
}
