package log_holdings

import (
	"context"
	"errors"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
)

func New(inner bonuses.Holdings, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  bonuses.Holdings
	logger *slog.Logger
}

var _ bonuses.Holdings = (*Logged)(nil)

func (l *Logged) Held(ctx context.Context, holder bonuses.Holder) (bonuses.Held, error) {
	held, err := l.inner.Held(ctx, holder)
	if err != nil && !errors.Is(ctx.Err(), context.Canceled) {
		l.logger.Error("failed to read the charges of a player due a box, offering it nothing this sweep",
			slog.String("account", string(holder)), slog.Any("error", err))
	}
	return held, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
