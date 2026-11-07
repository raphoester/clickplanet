package log_banned

import (
	"context"
	"errors"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in send_message_usecase.In) (messages.Message, error)
}

func New(implementation UseCase, logger *slog.Logger) *Decorator {
	return &Decorator{implementation: implementation, logger: logger}
}

type Decorator struct {
	implementation UseCase
	logger         *slog.Logger
}

// The sender is told an invalid message, as for any other refusal: the log says which phrase it was, and whose. Never
// the address, nor the text: the phrase listed is enough to read the list by, and a raid repeats it.
func (d *Decorator) Execute(ctx context.Context, in send_message_usecase.In) (messages.Message, error) {
	message, err := d.implementation.Execute(ctx, in)
	if errors.Is(err, messages.ErrBanned) {
		d.logger.Warn("chat message refused: a banned phrase",
			slog.String("account", in.Account.String()),
			slog.String("room", string(in.Room)),
			slog.Any("error", err),
		)
	}
	return message, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
