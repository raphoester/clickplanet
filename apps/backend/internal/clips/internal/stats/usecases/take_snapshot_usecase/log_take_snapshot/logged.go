package log_take_snapshot

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats/usecases/take_snapshot_usecase"
)

func New(inner take_snapshot_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  take_snapshot_usecase.Executor
	logger *slog.Logger
}

var _ take_snapshot_usecase.Executor = (*Logged)(nil)

// Counts and states, never a metric: the logs of a public repo are public.
func (l *Logged) Execute(ctx context.Context) (stats.Report, error) {
	report, err := l.inner.Execute(ctx)
	if err != nil {
		return report, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	for _, unread := range report.Unread {
		l.logger.Warn("a post of the job could not be read whole, and its snapshot is lost",
			slog.String("post", string(unread.PostID)), slog.String("reason", unread.Reason))
	}
	for _, lane := range report.TikTokAccounts {
		l.logger.Info("a tiktok account", slog.String("username", lane.Username), slog.String("type", lane.Type),
			slog.Bool("commercialMusic", lane.CommercialMusic), slog.Int("businessRows", lane.BusinessRows),
			slog.Bool("businessLaneLive", lane.Live()))
	}
	l.logger.Info("took a snapshot of the posts of the job", slog.Int("posts", report.Posts),
		slog.Int("notOut", report.NotOut), slog.Int("rows", report.RowCount()), slog.Int("syncing", report.Syncing))
	return report, nil
}
