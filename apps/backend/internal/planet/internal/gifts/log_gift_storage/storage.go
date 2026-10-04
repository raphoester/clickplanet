package log_gift_storage

import (
	"context"
	"errors"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/gifts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
)

func New(inner gifts.Storage, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  gifts.Storage
	logger *slog.Logger
}

var _ gifts.Storage = (*Logged)(nil)

func (l *Logged) Give(ctx context.Context, tag tempo.GiftTag, holder bonuses.Holder) error {
	err := l.inner.Give(ctx, tag, holder)
	if err != nil && !errors.Is(err, gifts.ErrGiven) && ctx.Err() == nil {
		l.logger.Error("failed to give a gift; the account's next click tries again",
			slog.String("tag", string(tag)), slog.String("account", string(holder)), slog.Any("error", err))
	}
	return err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
