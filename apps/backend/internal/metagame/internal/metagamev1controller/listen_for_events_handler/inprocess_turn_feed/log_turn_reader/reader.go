package log_turn_reader

import (
	"context"
	"log/slog"

	metagamev1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/metagame/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller/listen_for_events_handler/inprocess_turn_feed"
)

func New(inner inprocess_turn_feed.Reader, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  inprocess_turn_feed.Reader
	logger *slog.Logger
}

var _ inprocess_turn_feed.Reader = (*Logged)(nil)

func (l *Logged) Turn(ctx context.Context) (*metagamev1.Turn, error) {
	turn, err := l.inner.Turn(ctx)
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to read the live turn", slog.Any("error", err))
	}
	return turn, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
