package audit_approve_connection

import (
	"context"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/usecases/approve_connection_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in approve_connection_usecase.In) (approve_connection_usecase.Out, error)
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

// Every app let in is an app that can act as the operator: the line is the record of who let it in, and where.
func (a *Audited) Execute(ctx context.Context, in approve_connection_usecase.In) (approve_connection_usecase.Out, error) {
	started := time.Now()
	out, err := a.inner.Execute(ctx, in)

	attributes := []any{
		slog.String("operator", in.Operator.String()),
		slog.String("client", out.Client),
		slog.String("host", out.Host),
		slog.Duration("took", time.Since(started)),
	}
	if err != nil {
		a.logger.Warn("admin connection refused", append(attributes, slog.Any("error", err))...)
	} else {
		a.logger.Warn("admin connection approved", attributes...)
	}

	return out, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
