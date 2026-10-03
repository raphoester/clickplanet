package log_flush

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity/inmemory_event_buffer"
)

func New(inner inmemory_event_buffer.Flusher, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  inmemory_event_buffer.Flusher
	logger *slog.Logger
}

var _ inmemory_event_buffer.Flusher = (*Logged)(nil)

func (l *Logged) Flush(ctx context.Context) (inmemory_event_buffer.Flushed, error) {
	flushed, err := l.inner.Flush(ctx)

	switch {
	case err != nil:
		l.logger.Error("failed to flush the activity, retrying next tick",
			slog.Int("dropped", flushed.Dropped), slog.Any("error", err))
	case flushed.Dropped > 0:
		l.logger.Warn("the activity buffer was full, events dropped", slog.Int("dropped", flushed.Dropped))
	}

	return flushed, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
