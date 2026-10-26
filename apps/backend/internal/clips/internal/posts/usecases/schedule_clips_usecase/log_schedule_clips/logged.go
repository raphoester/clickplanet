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
		if outcome.Held {
			l.logger.Info("held a clip: every platform has posted its day", slog.String("clip", outcome.Title))
			continue
		}
		if outcome.Sound != nil {
			track := outcome.Sound.Track
			l.logger.Info("attached a trending track on tiktok", slog.String("clip", outcome.Title),
				slog.String("track", track.Title), slog.String("artist", track.Artist), slog.Int("rank", track.Rank),
				slog.String("chart", outcome.Sound.Country), slog.String("sound", track.ID))
		}
		if outcome.SoundMissed != "" {
			l.logger.Warn("a clip drawn for a trending track goes out with its anthem", slog.String("clip", outcome.Title),
				slog.String("why", outcome.SoundMissed))
		}
		for _, platform := range outcome.Platforms {
			attributes := []any{slog.String("clip", outcome.Title), slog.String("platform", string(platform.Platform)), slog.Time("at", outcome.At)}
			if platform.Failure != "" {
				l.logger.Error("failed to schedule a clip", append(attributes, slog.String("failure", platform.Failure))...)
				continue
			}
			l.logger.Info("scheduled a clip", append(attributes, slog.String("post", string(outcome.Post)))...)
		}
	}

	return report, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
