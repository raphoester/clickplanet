package audit_warn

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/warn_player_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in warn_player_usecase.In) (warn_player_usecase.Out, error)
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

func (a *Audited) Execute(ctx context.Context, in warn_player_usecase.In) (warn_player_usecase.Out, error) {
	out, err := a.inner.Execute(ctx, in)

	attrs := []any{
		slog.String("askedAccount", in.Account), slog.String("account", out.Account),
		slog.Time("warnedAt", out.WarnedAt), slog.Bool("alreadyWarned", out.AlreadyWarned),
	}

	if err != nil {
		a.logger.Warn("admin warning failed", append(attrs, slog.Any("error", err))...)
		return out, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin warning", attrs...)

	return out, nil
}
