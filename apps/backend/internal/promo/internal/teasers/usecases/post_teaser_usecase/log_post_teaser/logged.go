package log_post_teaser

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers/usecases/post_teaser_usecase"
)

func New(inner post_teaser_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  post_teaser_usecase.Executor
	logger *slog.Logger
}

var _ post_teaser_usecase.Executor = (*Logged)(nil)

// Each post and each platform that refused it; never its words.
func (l *Logged) Execute(ctx context.Context, battle teasers.BattleID, media teasers.Media) (teasers.Outcome, error) {
	outcome, err := l.inner.Execute(ctx, battle, media)
	if err != nil {
		return outcome, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}
	attributes := []any{slog.String("battle", string(battle))}
	switch outcome.Result {
	case teasers.Posted:
		for _, platform := range outcome.Platforms {
			on := append([]any{slog.String("platform", string(platform)), slog.Time("at", outcome.At)}, attributes...)
			if why, refused := outcome.Refused[platform]; refused {
				l.logger.Error("a platform refused a teaser", append(on, slog.String("why", why))...)
				continue
			}
			l.logger.Info("scheduled a teaser", append(on, slog.String("post", outcome.Post))...)
		}
	case teasers.Failed:
		l.logger.Error("a teaser was refused", append(attributes, slog.String("why", outcome.Why))...)
	default:
		l.logger.Warn("a teaser was "+string(outcome.Result), append(attributes, slog.String("why", outcome.Why))...)
	}
	return outcome, nil
}
