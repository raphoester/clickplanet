package log_verifier

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/ops/internal/access"
)

type Verifier interface {
	Caller(ctx context.Context, assertion string) (access.Caller, error)
}

func New(inner Verifier, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  Verifier
	logger *slog.Logger
}

func (l *Logged) Caller(ctx context.Context, assertion string) (access.Caller, error) {
	caller, err := l.inner.Caller(ctx, assertion)
	if err != nil {
		l.logger.Warn("ops refused a caller", slog.Any("error", err))
	}

	return caller, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
