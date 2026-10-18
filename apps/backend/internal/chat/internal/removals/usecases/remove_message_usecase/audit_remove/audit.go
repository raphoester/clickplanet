package audit_remove

import (
	"context"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type UseCase interface {
	Execute(ctx context.Context, id messages.MessageID) (time.Time, error)
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

func (a *Audited) Execute(ctx context.Context, id messages.MessageID) (time.Time, error) {
	removedAt, err := a.inner.Execute(ctx, id)
	if err != nil {
		a.logger.Warn("admin chat message removal failed", slog.String("message", string(id)), slog.Any("error", err))
		return removedAt, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin chat message removal", slog.String("message", string(id)))
	return removedAt, nil
}
