package log_admit

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admingate/admission"
)

type UseCase interface {
	Execute(ctx context.Context, visit admission.Visit) (admission.Operator, error)
}

func New(inner UseCase, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  UseCase
	logger *slog.Logger
}

func (l *Logged) Execute(ctx context.Context, visit admission.Visit) (admission.Operator, error) {
	operator, err := l.inner.Execute(ctx, visit)
	if err != nil {
		l.logger.Warn("admin call refused", slog.String("origin", visit.Origin), slog.Any("error", err))
	}

	return operator, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
