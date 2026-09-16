// Package audit_unban logs every lifted chat ban, refusals included.
package audit_unban

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans/usecases/unban_member_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in unban_member_usecase.In) (unban_member_usecase.Out, error)
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

// Execute logs at Warn, as the ban does: both halves of a moderation call are worth the same line.
func (a *Audited) Execute(
	ctx context.Context,
	in unban_member_usecase.In,
) (unban_member_usecase.Out, error) {
	out, err := a.inner.Execute(ctx, in)

	attrs := []any{slog.String("asked", in.AuthorTag), slog.String("authorTag", out.AuthorTag)}

	if err != nil {
		a.logger.Warn("chat unban failed", append(attrs, slog.Any("error", err))...)
		return out, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("chat unban", attrs...)

	return out, nil
}
