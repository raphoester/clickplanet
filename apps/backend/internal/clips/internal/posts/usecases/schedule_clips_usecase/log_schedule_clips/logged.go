package log_schedule_clips

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/schedule_clips_usecase"
)

func New(inner schedule_clips_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  schedule_clips_usecase.Executor
	logger *slog.Logger
}

var _ schedule_clips_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) (posts.Report, error) {
	report, err := l.inner.Execute(ctx)

	for _, outcome := range report {
		if outcome.Failure != "" {
			l.logger.Error("failed to schedule a clip",
				slog.String("clip", outcome.Title), slog.Time("at", outcome.At), slog.String("failure", outcome.Failure))
			continue
		}
		l.logger.Info("scheduled a clip",
			slog.String("clip", outcome.Title), slog.Time("at", outcome.At), slog.String("post", string(outcome.Post)))
	}

	return report, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
