package audit_save_rules

import (
	"context"
	"log/slog"

	"google.golang.org/protobuf/encoding/protojson"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
)

type UseCase interface {
	Execute(ctx context.Context, in *missionsv1.Rules) error
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

func (a *Audited) Execute(ctx context.Context, in *missionsv1.Rules) error {
	err := a.inner.Execute(ctx, in)
	written, _ := protojson.Marshal(in)
	attrs := []any{slog.String("rules", string(written))}
	if err != nil {
		a.logger.Warn("admin missions rules refused", append(attrs, slog.Any("error", err))...)
		return err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}
	a.logger.Warn("admin missions rules saved", attrs...)
	return nil
}
