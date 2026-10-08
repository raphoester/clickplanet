package log_race_reader

import (
	"context"
	"log/slog"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/listen_for_events_handler/inprocess_race_feed"
)

func New(inner inprocess_race_feed.Reader, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  inprocess_race_feed.Reader
	logger *slog.Logger
}

var _ inprocess_race_feed.Reader = (*Logged)(nil)

func (l *Logged) Race(ctx context.Context) (*seasonsv1.Race, error) {
	race, err := l.inner.Race(ctx)
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to read the live race", slog.Any("error", err))
	}
	return race, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
