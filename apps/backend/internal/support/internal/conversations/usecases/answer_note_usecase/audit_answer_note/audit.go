package audit_answer_note

import (
	"context"
	"log/slog"
	"time"
	"unicode/utf8"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations/usecases/answer_note_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in answer_note_usecase.In) (time.Time, error)
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

func (a *Audited) Execute(ctx context.Context, in answer_note_usecase.In) (time.Time, error) {
	attributes := []any{
		slog.String("note", in.Note.String()),
		slog.Int("characters", utf8.RuneCountInString(in.Text)),
		slog.String("client", cpctx.GetOperator(ctx).Client()),
	}

	sentAt, err := a.inner.Execute(ctx, in)
	if err != nil {
		a.logger.Warn("admin note answer failed", append(attributes, slog.Any("error", err))...)
		return sentAt, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin note answer", attributes...)
	return sentAt, nil
}
