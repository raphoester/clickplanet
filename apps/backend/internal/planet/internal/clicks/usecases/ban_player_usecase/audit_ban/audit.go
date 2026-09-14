// Package audit_ban logs every operator ban, refusals included.
package audit_ban

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/ban_player_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in ban_player_usecase.In) (ban_player_usecase.Out, error)
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

// Execute logs at Warn: the log is the only record that a person, not a watchdog, passed this ban.
func (a *Audited) Execute(ctx context.Context, in ban_player_usecase.In) (ban_player_usecase.Out, error) {
	out, err := a.inner.Execute(ctx, in)

	attrs := []any{
		slog.String("asked", in.Scope), slog.Duration("duration", in.Duration),
		slog.String("scope", out.Scope), slog.Int("offence", out.Offence),
		slog.Time("bannedUntil", out.Until), slog.Bool("enforced", out.Enforced),
	}

	if err != nil {
		a.logger.Warn("admin ban failed", append(attrs, slog.Any("error", err))...)
		return out, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin ban", attrs...)

	return out, nil
}
