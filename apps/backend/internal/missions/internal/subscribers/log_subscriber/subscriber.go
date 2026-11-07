package log_subscriber

import (
	"context"
	"log/slog"

	"google.golang.org/protobuf/proto"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

func New[T proto.Message](inner cpredis.Handler[T], logger *slog.Logger) *Logged[T] {
	return &Logged[T]{inner: inner, logger: logger}
}

type Logged[T proto.Message] struct {
	inner  cpredis.Handler[T]
	logger *slog.Logger
}

func (l *Logged[T]) Handle(ctx context.Context, event T) error {
	err := l.inner.Handle(ctx, event)
	if err != nil {
		l.logger.Error(
			"the missions module refused an event",
			slog.String("event", string(event.ProtoReflect().Descriptor().FullName())),
			slog.Any("error", err),
		)
	}
	return err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}

func NewBatch[T proto.Message](inner cpredis.BatchHandler[T], logger *slog.Logger) *LoggedBatch[T] {
	return &LoggedBatch[T]{inner: inner, logger: logger}
}

type LoggedBatch[T proto.Message] struct {
	inner  cpredis.BatchHandler[T]
	logger *slog.Logger
}

func (l *LoggedBatch[T]) HandleBatch(ctx context.Context, events []T) error {
	err := l.inner.HandleBatch(ctx, events)
	if err != nil {
		var zero T
		l.logger.Error(
			"the missions module refused a batch of events",
			slog.String("event", string(zero.ProtoReflect().Descriptor().FullName())),
			slog.Int("events", len(events)),
			slog.Any("error", err),
		)
	}
	return err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
