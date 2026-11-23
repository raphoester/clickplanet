package log_fetch_flag_list

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_flag_list_usecase"
)

func New(inner fetch_flag_list_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  fetch_flag_list_usecase.Executor
	logger *slog.Logger
}

var _ fetch_flag_list_usecase.Executor = (*Logged)(nil)

// A flag list left out is a warning, not a failure: a sold flag is named by its code and drawn blank.
func (l *Logged) Execute(ctx context.Context) (replays.FlagList, error) {
	list, err := l.inner.Execute(ctx)
	if err != nil {
		l.logger.Warn("left the flag list out: a sold flag is named by its code and drawn blank", slog.Any("error", err))
		return list, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}
	l.logger.Info("saved the flag list", slog.Int("sold", list.Sold))
	return list, nil
}
