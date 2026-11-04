package log_link_members

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/usecases/link_members_usecase"
)

func New(inner link_members_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  link_members_usecase.Executor
	logger *slog.Logger
}

var _ link_members_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) (link_members_usecase.Out, error) {
	out, err := l.inner.Execute(ctx)
	for _, failed := range out.Failed {
		l.logger.Warn("failed to link a Discord member", slog.String("account", failed.Account.String()),
			slog.String("discord_user", string(failed.User)), slog.Any("error", failed.Err))
	}
	attrs := []any{slog.Int("linked", out.Linked), slog.Int("outside", out.Outside), slog.Int("failed", len(out.Failed))}
	if err != nil {
		l.logger.Error("failed to link the Discord members", append(attrs, slog.Any("error", err))...)
	} else {
		l.logger.Info("Discord members linked", attrs...)
	}
	return out, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
