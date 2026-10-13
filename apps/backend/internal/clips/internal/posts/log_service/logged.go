package log_service

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

func New(inner posts.Service, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  posts.Service
	logger *slog.Logger
}

var _ posts.Service = (*Logged)(nil)

func (l *Logged) Schedule(ctx context.Context, post posts.Post) (posts.PostID, error) {
	id, err := l.inner.Schedule(ctx, post)
	if err != nil {
		l.logger.Warn("the posting service did not take a clip",
			slog.String("clip", post.TakeSnapshot().Title), slog.Any("error", err))
	}

	return id, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
