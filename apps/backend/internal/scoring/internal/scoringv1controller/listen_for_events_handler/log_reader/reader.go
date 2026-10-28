package log_reader

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/protobuf/proto"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/listen_for_events_handler/inprocess_feed"
)

func New[K comparable, V proto.Message](what string, inner inprocess_feed.Reader[K, V], logger *slog.Logger) *Logged[K, V] {
	return &Logged[K, V]{what: what, inner: inner, logger: logger}
}

type Logged[K comparable, V proto.Message] struct {
	what   string
	inner  inprocess_feed.Reader[K, V]
	logger *slog.Logger
}

func (l *Logged[K, V]) Live(ctx context.Context, key K) (V, bool, error) {
	value, shows, err := l.inner.Live(ctx, key)
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to read a live "+l.what, slog.String("key", fmt.Sprint(key)), slog.Any("error", err))
	}
	return value, shows, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
