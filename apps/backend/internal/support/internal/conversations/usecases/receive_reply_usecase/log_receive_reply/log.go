package log_receive_reply

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations/usecases/receive_reply_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

func New(inner receive_reply_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  receive_reply_usecase.Executor
	logger *slog.Logger
}

func (l *Logged) Execute(ctx context.Context, in receive_reply_usecase.In) (notes.NoteID, error) {
	note, err := l.inner.Execute(ctx, in)
	attributes := []any{slog.String("recipient", in.Recipient), slog.String("note", note.String())}
	switch {
	case err == nil:
		l.logger.Info("a reply to a note was kept", attributes...)
	case receive_reply_usecase.IsDropped(err):
		l.logger.Warn("a reply to a note was dropped", append(attributes, slog.Any("reason", err))...)
	default:
		l.logger.Error("a reply to a note could not be kept", append(attributes, slog.Any("error", err))...)
	}
	return note, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
