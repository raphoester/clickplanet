package audit_decide_report

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/decisions/usecases/decide_report_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in decide_report_usecase.In) (time.Time, error)
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

func (a *Audited) Execute(ctx context.Context, in decide_report_usecase.In) (time.Time, error) {
	attributes := []any{
		slog.String("report", uuid.UUID(in.Report).String()),
		slog.String("outcome", string(in.Outcome)),
		slog.String("comment", in.Comment),
	}

	decidedAt, err := a.inner.Execute(ctx, in)
	if err != nil {
		a.logger.Warn("admin report decision failed", append(attributes, slog.Any("error", err))...)
		return decidedAt, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin report decision", attributes...)
	return decidedAt, nil
}
