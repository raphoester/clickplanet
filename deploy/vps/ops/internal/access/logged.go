package access

import (
	"context"
	"log/slog"
)

func NewLogged(inner Verifier, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  Verifier
	logger *slog.Logger
}

func (l *Logged) Verify(ctx context.Context, assertion string) (Caller, error) {
	caller, err := l.inner.Verify(ctx, assertion)
	if err != nil {
		l.logger.Warn("refused a caller", slog.Any("error", err))
	}
	return caller, err
}
