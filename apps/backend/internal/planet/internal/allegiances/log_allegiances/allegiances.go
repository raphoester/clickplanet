package log_allegiances

import (
	"context"
	"errors"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Allegiances interface {
	PlayError(ctx context.Context, payer clicks.Payer, flag string) error
}

func New(inner Allegiances, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  Allegiances
	logger *slog.Logger
}

func (l *Logged) PlayError(ctx context.Context, payer clicks.Payer, flag string) error {
	err := l.inner.PlayError(ctx, payer, flag)
	if errors.Is(err, allegiances.ErrUnread) {
		l.logger.Error("a flag could not be read, so the caller plays on", slog.Any("error", err))
	}
	return err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
