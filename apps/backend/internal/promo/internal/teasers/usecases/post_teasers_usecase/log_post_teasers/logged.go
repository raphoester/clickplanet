package log_post_teasers

import (
	"context"
	"log/slog"
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers/usecases/post_teasers_usecase"
)

func New(inner post_teasers_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  post_teasers_usecase.Executor
	logger *slog.Logger
}

var _ post_teasers_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) (teasers.Report, error) {
	report, err := l.inner.Execute(ctx)
	for _, outcome := range report {
		attributes := []any{slog.String("battle", string(outcome.Battle)), slog.String("teaser", outcome.Title)}
		switch outcome.Result {
		case teasers.Scheduled:
			for _, platform := range outcome.Platforms {
				on := slices.Concat(attributes, []any{slog.String("platform", string(platform)), slog.Time("at", outcome.At)})
				if refused, ok := outcome.Refused[platform]; ok {
					l.logger.Error("a platform refused a teaser", append(on, slog.String("why", refused))...)
					continue
				}
				l.logger.Info("scheduled a teaser", append(on, slog.String("post", outcome.Post))...)
			}
		case teasers.Failed:
			l.logger.Error("failed to schedule a teaser", append(attributes, slog.String("why", outcome.Why))...)
		default:
			l.logger.Warn("a teaser was "+string(outcome.Result), append(attributes, slog.String("why", outcome.Why))...)
		}
	}
	return report, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
