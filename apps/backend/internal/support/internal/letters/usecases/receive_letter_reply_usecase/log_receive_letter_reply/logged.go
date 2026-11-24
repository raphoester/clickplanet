package log_receive_letter_reply

import (
	"context"
	"errors"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters/usecases/receive_letter_reply_usecase"
)

func New(inner receive_letter_reply_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  receive_letter_reply_usecase.Executor
	logger *slog.Logger
}

func (l *Logged) Execute(ctx context.Context, in receive_letter_reply_usecase.In) (letters.LetterID, error) {
	letter, err := l.inner.Execute(ctx, in)
	attributes := []any{slog.String("recipient", in.Recipient), slog.String("letter", letter.String())}
	switch {
	case errors.Is(err, letters.ErrNotALetter):
	case err == nil:
		l.logger.Info("a reply to a letter was kept", attributes...)
	case errors.Is(err, conversations.ErrReceivedAlready):
		l.logger.Info("a reply to a letter was kept already", attributes...)
	case receive_letter_reply_usecase.IsStray(err):
		l.logger.Warn("a mail no letter keeps goes to the operator's mailbox", append(attributes, slog.Any("reason", err))...)
	default:
		l.logger.Error("a reply to a letter could not be kept", append(attributes, slog.Any("error", err))...)
	}
	return letter, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
