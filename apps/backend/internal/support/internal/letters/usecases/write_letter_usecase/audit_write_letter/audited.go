package audit_write_letter

import (
	"context"
	"log/slog"
	"unicode/utf8"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters/usecases/write_letter_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in write_letter_usecase.In) (letters.Snapshot, error)
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

func (a *Audited) Execute(ctx context.Context, in write_letter_usecase.In) (letters.Snapshot, error) {
	attributes := []any{
		slog.String("account", in.Account),
		slog.Int("characters", utf8.RuneCountInString(in.Text)),
		slog.String("client", cpctx.GetOperator(ctx).Client()),
	}
	written, err := a.inner.Execute(ctx, in)
	attributes = append(attributes, slog.String("letter", written.ID.String()))
	if err != nil {
		a.logger.Warn("admin letter failed", append(attributes, slog.Any("error", err))...)
		return written, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}
	a.logger.Warn("admin letter", attributes...)
	return written, nil
}
