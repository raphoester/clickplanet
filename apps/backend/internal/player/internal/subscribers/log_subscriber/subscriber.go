// Package log_subscriber logs an event a subscriber refused. The bus only counts it.
package log_subscriber

import (
	"context"
	"log/slog"

	"google.golang.org/protobuf/proto"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

func New[T proto.Message](inner cpbootstrap.Handler[T], logger *slog.Logger) *Logged[T] {
	return &Logged[T]{inner: inner, logger: logger}
}

type Logged[T proto.Message] struct {
	inner  cpbootstrap.Handler[T]
	logger *slog.Logger
}

func (l *Logged[T]) Handle(ctx context.Context, event T) error {
	err := l.inner.Handle(ctx, event)
	if err != nil {
		l.logger.Error("the player module refused an event",
			slog.String("event", string(event.ProtoReflect().Descriptor().FullName())),
			slog.Any("error", err),
		)
	}
	return err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
