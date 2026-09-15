// Package audit_reassign logs every reassignment, dry runs and failures included.
package audit_reassign

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/reassign_country_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in reassign_country_usecase.In) (reassign_country_usecase.Out, error)
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

// Execute logs at Warn: the log is the only record that these tiles did not change hands through play.
func (a *Audited) Execute(ctx context.Context, in reassign_country_usecase.In) (reassign_country_usecase.Out, error) {
	out, err := a.inner.Execute(ctx, in)

	attrs := []any{
		slog.String("from", in.From), slog.String("to", in.To), slog.Bool("dryRun", in.DryRun),
		slog.Int("fromBefore", out.FromBefore), slog.Int("toBefore", out.ToBefore),
		slog.Int("moved", out.Moved), slog.Int("fromAfter", out.FromAfter), slog.Int("toAfter", out.ToAfter),
	}

	if err != nil {
		a.logger.Warn("admin country reassignment failed", append(attrs, slog.Any("error", err))...)
		return out, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin country reassignment", attrs...)

	return out, nil
}
