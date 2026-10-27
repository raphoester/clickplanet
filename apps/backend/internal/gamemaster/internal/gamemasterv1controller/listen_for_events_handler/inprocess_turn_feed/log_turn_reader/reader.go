package log_turn_reader

import (
	"context"
	"log/slog"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/listen_for_events_handler/inprocess_turn_feed"
)

func New(inner inprocess_turn_feed.Reader, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  inprocess_turn_feed.Reader
	logger *slog.Logger
}

var _ inprocess_turn_feed.Reader = (*Logged)(nil)

func (l *Logged) Now(ctx context.Context) (*gamemasterv1.GetTurnResponse, error) {
	now, err := l.inner.Now(ctx)
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to read the live turn", slog.Any("error", err))
	}
	return now, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
