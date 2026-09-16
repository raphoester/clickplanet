// Package audit_ban logs every chat ban, refusals included.
package audit_ban

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans/usecases/ban_member_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in ban_member_usecase.In) (ban_member_usecase.Out, error)
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

// Execute logs at Warn: the log is the only record that a person silenced this member, and why.
func (a *Audited) Execute(
	ctx context.Context,
	in ban_member_usecase.In,
) (ban_member_usecase.Out, error) {
	out, err := a.inner.Execute(ctx, in)

	attrs := []any{
		slog.String("asked", in.AuthorTag), slog.String("reason", in.Reason),
		slog.String("authorTag", out.Ban.AuthorTag), slog.Time("bannedAt", out.Ban.BannedAt),
		slog.Int("redacted", out.Redacted),
	}

	if err != nil {
		a.logger.Warn("chat ban failed", append(attrs, slog.Any("error", err))...)
		return out, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("chat ban", attrs...)

	return out, nil
}
