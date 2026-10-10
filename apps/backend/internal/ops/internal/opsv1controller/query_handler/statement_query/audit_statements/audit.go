package audit_statements

import (
	"context"
	"log/slog"

	opsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/ops/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/ops/internal/access"
)

type Query interface {
	Rows(ctx context.Context, statement string, limit uint32) (*opsv1.QueryResponse, error)
}

func New(inner Query, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  Query
	logger *slog.Logger
}

func (a *Audited) Rows(ctx context.Context, statement string, limit uint32) (*opsv1.QueryResponse, error) {
	rows, err := a.inner.Rows(ctx, statement, limit)

	attrs := []any{
		slog.String("caller", string(access.CallerOf(ctx))),
		slog.String("statement", statement),
	}

	if err != nil {
		a.logger.Warn("ops statement failed", append(attrs, slog.Any("error", err))...)
		return rows, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Info("ops statement", append(attrs,
		slog.Int("rows", len(rows.GetRows())), slog.Bool("truncated", rows.GetTruncated()),
	)...)

	return rows, nil
}
