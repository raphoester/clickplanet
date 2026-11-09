package log_schedule_post

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts/usecases/schedule_post_usecase"
)

func New(inner schedule_post_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

// Each post with its title, its time and its platforms, never its words or its metadata.
type Logged struct {
	inner  schedule_post_usecase.Executor
	logger *slog.Logger
}

var _ schedule_post_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context, post posts.Post) schedule_post_usecase.Outcome {
	outcome := l.inner.Execute(ctx, post)

	snapshot := post.TakeSnapshot()
	attributes := []any{slog.String("title", snapshot.Title), slog.Time("at", snapshot.At), slog.Any("platforms", snapshot.Platforms)}
	switch {
	case outcome.Scheduled != nil:
		l.logger.Info("scheduled a post", append(attributes, slog.String("post", string(outcome.Scheduled.ID)))...)
		for platform, why := range outcome.Scheduled.Refused {
			l.logger.Warn("a platform refused a post", append(attributes, slog.String("platform", string(platform)),
				slog.String("why", why.Error()))...)
		}
	case outcome.Busy != nil:
		l.logger.Warn("the posting service asked to try a post later", append(attributes,
			slog.Duration("retry_after", outcome.Busy.RetryAfter), slog.String("why", outcome.Busy.Reason))...)
	case outcome.Taken != "":
		l.logger.Info("a post's slot was taken", append(attributes, slog.String("why", outcome.Taken))...)
	default:
		l.logger.Error("failed to schedule a post", append(attributes, slog.String("why", outcome.Failed))...)
	}
	return outcome
}
