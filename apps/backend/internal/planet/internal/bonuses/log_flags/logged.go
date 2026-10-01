// Package log_flags logs a failed read of the flags the bonus bands are drawn from: the sweep only skips its offers.
package log_flags

import (
	"context"
	"errors"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func New(inner bonuses.Flags, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  bonuses.Flags
	logger *slog.Logger
}

var _ bonuses.Flags = (*Logged)(nil)

// Allegiances logs a failure at Error, unless the process is stopping.
func (l *Logged) Allegiances(
	ctx context.Context, keys ...clicks.AllegianceKey,
) (map[clicks.AllegianceKey]clicks.Allegiance, error) {
	tallies, err := l.inner.Allegiances(ctx, keys...)
	if err != nil && !errors.Is(ctx.Err(), context.Canceled) {
		l.logger.Error("failed to read the players' flags, no box or quiz this sweep",
			slog.Int("keys", len(keys)), slog.Any("error", err))
	}

	return tallies, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
