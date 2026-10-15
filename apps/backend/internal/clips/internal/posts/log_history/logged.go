package log_history

import (
	"context"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/choose_clips_usecase"
)

func New(inner choose_clips_usecase.History, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  choose_clips_usecase.History
	logger *slog.Logger
}

var _ choose_clips_usecase.History = (*Logged)(nil)

func (l *Logged) Told(ctx context.Context, since time.Time) ([]posts.Telling, error) {
	told, err := l.inner.Told(ctx, since)

	for _, telling := range told {
		attributes := []any{
			slog.String("clip", telling.Title), slog.Time("at", telling.At), slog.String("state", telling.State),
			slog.String("flag", telling.Key.Flag), slog.String("continent", telling.Key.Continent),
		}
		if telling.Failure != "" {
			l.logger.Warn("a post of the last day was refused", append(attributes, slog.String("failure", telling.Failure))...)
			continue
		}
		l.logger.Info("a post of the last day", attributes...)
	}

	return told, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
