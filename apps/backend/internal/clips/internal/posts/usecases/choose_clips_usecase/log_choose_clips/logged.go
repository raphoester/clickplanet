package log_choose_clips

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/choose_clips_usecase"
)

func New(inner choose_clips_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  choose_clips_usecase.Executor
	logger *slog.Logger
}

var _ choose_clips_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) (posts.Shortlist, error) {
	chosen, err := l.inner.Execute(ctx)

	switch {
	case err != nil:
	case len(chosen) == 0:
		l.logger.Info("no story is worth a clip")
	default:
		for _, story := range chosen {
			l.logger.Info("chose a story", slog.Int("pick", int(story.Pick)), slog.String("title", story.Title))
		}
	}

	return chosen, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
