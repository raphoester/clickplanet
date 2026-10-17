package audit_name_accounts

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/name_accounts_usecase"
)

func New(inner name_accounts_usecase.Executor, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  name_accounts_usecase.Executor
	logger *slog.Logger
}

var _ name_accounts_usecase.Executor = (*Audited)(nil)

func (a *Audited) Execute(ctx context.Context) (int, error) {
	named, err := a.inner.Execute(ctx)
	if err != nil {
		a.logger.Warn("admin account naming failed", slog.Int("named", named), slog.Any("error", err))
		return named, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}
	a.logger.Warn("admin account naming", slog.Int("named", named))
	return named, nil
}
