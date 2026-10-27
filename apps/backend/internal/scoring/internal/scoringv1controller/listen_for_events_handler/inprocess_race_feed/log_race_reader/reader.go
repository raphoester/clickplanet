package log_race_reader

import (
	"context"
	"log/slog"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/listen_for_events_handler/inprocess_race_feed"
)

func New(inner inprocess_race_feed.Reader, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  inprocess_race_feed.Reader
	logger *slog.Logger
}

var _ inprocess_race_feed.Reader = (*Logged)(nil)

func (l *Logged) Race(ctx context.Context) (*scoringv1.Race, error) {
	race, err := l.inner.Race(ctx)
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to read the live race", slog.Any("error", err))
	}
	return race, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
