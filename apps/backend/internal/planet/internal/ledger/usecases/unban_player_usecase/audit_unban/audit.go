package audit_unban

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/unban_player_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in unban_player_usecase.In) (unban_player_usecase.Out, error)
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

func (a *Audited) Execute(ctx context.Context, in unban_player_usecase.In) (unban_player_usecase.Out, error) {
	out, err := a.inner.Execute(ctx, in)

	attrs := []any{
		slog.String("asked", in.Scope), slog.String("askedAccount", in.Account),
		slog.String("scope", out.Scope), slog.String("account", out.Account), slog.Int("offence", out.Offence),
		slog.Time("bannedUntil", out.Until),
	}

	if err != nil {
		a.logger.Warn("admin unban failed", append(attrs, slog.Any("error", err))...)
		return out, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin unban", attrs...)

	return out, nil
}
