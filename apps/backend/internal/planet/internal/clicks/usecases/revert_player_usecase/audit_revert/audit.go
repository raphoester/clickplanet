// Package audit_revert logs every operator revert, dry runs and failures included.
package audit_revert

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/revert_player_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in revert_player_usecase.In) (revert_player_usecase.Out, error)
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

// Execute logs at Warn: the log is the only record that these tiles did not change hands through play.
func (a *Audited) Execute(ctx context.Context, in revert_player_usecase.In) (revert_player_usecase.Out, error) {
	out, err := a.inner.Execute(ctx, in)

	attrs := []any{
		slog.String("asked", in.Scope), slog.String("scope", out.Scope), slog.Bool("dryRun", in.DryRun),
		slog.Int("touched", out.Touched), slog.Int("held", out.Held), slog.Int("restored", out.Restored),
	}

	if err != nil {
		a.logger.Warn("admin player revert failed", append(attrs, slog.Any("error", err))...)
		return out, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin player revert", attrs...)

	return out, nil
}
