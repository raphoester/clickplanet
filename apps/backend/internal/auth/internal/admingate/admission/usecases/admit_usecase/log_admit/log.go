package log_admit

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admingate/admission"
)

type UseCase interface {
	Execute(ctx context.Context, visit admission.Visit) (admission.Caller, error)
}

func New(inner UseCase, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  UseCase
	logger *slog.Logger
}

// Never the cookie, and never the token: whether a token came is all the line says of it.
func (l *Logged) Execute(ctx context.Context, visit admission.Visit) (admission.Caller, error) {
	caller, err := l.inner.Execute(ctx, visit)
	if err != nil {
		l.logger.Warn("admin call refused",
			slog.String("origin", visit.Origin),
			slog.Bool("token", visit.Token != ""),
			slog.Any("error", err),
		)
	}

	return caller, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
