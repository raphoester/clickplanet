package cprules

import (
	"context"
	"log/slog"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type Saving[M proto.Message] interface {
	Execute(ctx context.Context, in Snapshot[M]) error
}

// SaveLog is what AuditedSave says at Warn, in the module's words: Saved with each set kept, Refused with each one
// refused.
type SaveLog struct {
	Saved   string
	Refused string
}

func NewAuditedSave[M proto.Message](inner Saving[M], logger *slog.Logger, says SaveLog) *AuditedSave[M] {
	return &AuditedSave[M]{inner: inner, logger: logger, says: says}
}

type AuditedSave[M proto.Message] struct {
	inner  Saving[M]
	logger *slog.Logger
	says   SaveLog
}

func (a *AuditedSave[M]) Execute(ctx context.Context, in Snapshot[M]) error {
	err := a.inner.Execute(ctx, in)

	written, _ := protojson.Marshal(in.Rules)
	attrs := []any{slog.String("set", string(in.Name)), slog.String("rules", string(written))}

	if err != nil {
		a.logger.Warn(a.says.Refused, append(attrs, slog.Any("error", err))...)
		return err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn(a.says.Saved, attrs...)

	return nil
}
