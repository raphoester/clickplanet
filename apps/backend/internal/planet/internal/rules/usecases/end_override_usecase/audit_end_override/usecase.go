package audit_end_override

import (
	"context"
	"log/slog"
)

type UseCase interface {
	Execute(ctx context.Context) error
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

func (a *Audited) Execute(ctx context.Context) error {
	if err := a.inner.Execute(ctx); err != nil {
		a.logger.Warn("admin rule override end refused", slog.Any("error", err))
		return err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin rule override ended")

	return nil
}
