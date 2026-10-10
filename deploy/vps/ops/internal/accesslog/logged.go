package accesslog

import (
	"context"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-ops/internal/access"
	"github.com/raphoester/clickplanet.lol-ops/internal/excerpt"
)

func NewLogged(inner Reader, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  Reader
	logger *slog.Logger
}

func (l *Logged) Read(ctx context.Context, query Query) (excerpt.Excerpt, error) {
	start := time.Now()
	found, err := l.inner.Read(ctx, query)

	attributes := []any{
		slog.String("caller", string(access.CallerFrom(ctx))),
		slog.Time("since", query.Since),
		slog.Time("until", query.Until),
		slog.String("contains", query.Contains),
		slog.Duration("took", time.Since(start)),
	}
	if err != nil {
		l.logger.Warn("an access log read failed", append(attributes, slog.Any("error", err))...)
		return found, err
	}
	l.logger.Info("read the access log", append(
		attributes,
		slog.Int("lines", len(found.Lines)),
		slog.Bool("truncated", found.Truncated),
	)...)
	return found, nil
}
