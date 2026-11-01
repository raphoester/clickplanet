package audit_release

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
)

type Book interface {
	Release(ctx context.Context, account allegiances.Account) error
}

func New(inner Book, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  Book
	logger *slog.Logger
}

func (a *Audited) Release(ctx context.Context, account allegiances.Account) error {
	err := a.inner.Release(ctx, account)

	attrs := []any{slog.String("account", string(account))}
	if err != nil {
		a.logger.Warn("admin allegiance release failed", append(attrs, slog.Any("error", err))...)
		return err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin allegiance release", attrs...)

	return nil
}
