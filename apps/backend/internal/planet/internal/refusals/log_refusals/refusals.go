package log_refusals

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Refusals interface {
	Refusal(ctx context.Context, payer clicks.Payer) (clicks.Refusal, bool, error)
}

func New(inner Refusals, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  Refusals
	logger *slog.Logger
}

func (l *Logged) Refusal(ctx context.Context, payer clicks.Payer) (clicks.Refusal, bool, error) {
	refusal, refused, err := l.inner.Refusal(ctx, payer)
	if err != nil {
		l.logger.Error("a refusal could not be read, so the caller plays on", slog.Any("error", err))
	}
	return refusal, refused, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
