package log_keep_appearances

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/keep_appearances_usecase"
)

func New(inner keep_appearances_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  keep_appearances_usecase.Executor
	logger *slog.Logger
}

var _ keep_appearances_usecase.Executor = (*Logged)(nil)

// Counts and the post, never a name: the logs of a public repo are public, and the names are in the caption already.
func (l *Logged) Execute(ctx context.Context, report posts.Report) []keep_appearances_usecase.Kept {
	kept := l.inner.Execute(ctx, report)
	for _, one := range kept {
		attributes := []any{
			slog.String("post", string(one.Appearance.Post)), slog.Int("fighters", len(one.Appearance.Fighters)),
		}
		if one.Err != nil {
			l.logger.Warn("the players a post names will not be told in the game", append(attributes, slog.String("why", one.Err.Error()))...)
			continue
		}
		l.logger.Info("the players a post names will be told in the game", append(attributes, slog.Int("named", one.Named))...)
	}
	return kept
}
