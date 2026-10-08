package log_board_reader

import (
	"context"
	"log/slog"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/listen_for_events_handler/inprocess_board_feed"
)

func New(inner inprocess_board_feed.Reader, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  inprocess_board_feed.Reader
	logger *slog.Logger
}

var _ inprocess_board_feed.Reader = (*Logged)(nil)

func (l *Logged) Board(ctx context.Context, country string) (*seasonsv1.Board, error) {
	board, err := l.inner.Board(ctx, country)
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to read a live board", slog.String("country", country), slog.Any("error", err))
	}
	return board, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
