package audit_grant_charges

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/grant_charges_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in grant_charges_usecase.In) (grant_charges_usecase.Out, error)
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

func (a *Audited) Execute(ctx context.Context, in grant_charges_usecase.In) (grant_charges_usecase.Out, error) {
	out, err := a.inner.Execute(ctx, in)

	attrs := []any{
		slog.String("asked", in.Account),
		slog.Bool("refill", in.Grant.Refill), slog.Bool("bomb", in.Grant.Bomb),
		slog.Int("enclosures", in.Grant.Enclosures), slog.Int("spreadClicks", in.Grant.SpreadClicks),
		slog.Int("defenders", in.Grant.Defenders),
		slog.String("account", string(out.Holder)),
		slog.Any("before", out.Before), slog.Any("after", out.After),
	}

	if err != nil {
		a.logger.Warn("admin charge grant failed", append(attrs, slog.Any("error", err))...)
		return out, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin charge grant", attrs...)

	return out, nil
}
