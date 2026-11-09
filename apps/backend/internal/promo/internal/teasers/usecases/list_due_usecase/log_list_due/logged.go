package log_list_due

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers/usecases/list_due_usecase"
)

func New(inner list_due_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  list_due_usecase.Executor
	logger *slog.Logger
}

var _ list_due_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) ([]list_due_usecase.Due, error) {
	due, err := l.inner.Execute(ctx)
	if err != nil {
		return due, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}
	if len(due) == 0 {
		l.logger.Info("no battle is due its teaser: none starts within 36 hours, or each is teased already")
	}
	for _, one := range due {
		l.logger.Info("a battle is due its teaser", slog.String("battle", string(one.Teaser.Battle)),
			slog.Time("starts", one.Teaser.StartsAt), slog.Int("cards", len(one.Teaser.Cards)), slog.String("folder", one.Folder))
	}
	return due, nil
}
