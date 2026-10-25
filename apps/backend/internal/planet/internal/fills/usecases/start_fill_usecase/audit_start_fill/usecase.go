package audit_start_fill

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills/usecases/start_fill_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in start_fill_usecase.In) error
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

func (a *Audited) Execute(ctx context.Context, in start_fill_usecase.In) error {
	err := a.inner.Execute(ctx, in)

	attrs := []any{slog.String("request", string(in.Request)), slog.String("source", string(in.Source))}

	if err != nil {
		a.logger.Warn("charge fill refused", append(attrs, slog.Any("error", err))...)
		return err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("charge fill asked", attrs...)

	return nil
}
