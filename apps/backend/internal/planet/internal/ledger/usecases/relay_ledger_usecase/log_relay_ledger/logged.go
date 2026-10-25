package log_relay_ledger

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/relay_ledger_usecase"
)

func New(inner relay_ledger_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  relay_ledger_usecase.Executor
	logger *slog.Logger
}

var _ relay_ledger_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) (int, error) {
	told, err := l.inner.Execute(ctx)
	if err != nil {
		l.logger.Error("failed to tell what the ledger saved, retrying next tick", slog.Any("error", err))
	}
	return told, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
