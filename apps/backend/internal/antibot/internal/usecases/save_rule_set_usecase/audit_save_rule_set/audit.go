package audit_save_rule_set

import (
	"context"
	"log/slog"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
)

type UseCase interface {
	Execute(ctx context.Context, in rules.Snapshot) error
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

func (a *Audited) Execute(ctx context.Context, in rules.Snapshot) error {
	err := a.inner.Execute(ctx, in)

	written, _ := protojson.Marshal(in.Rules)
	attrs := []any{slog.String("set", string(in.Name)), slog.String("rules", string(written))}

	if err != nil {
		a.logger.Warn("admin antibot rule set refused", append(attrs, slog.Any("error", err))...)
		return err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin antibot rule set saved", attrs...)

	return nil
}
