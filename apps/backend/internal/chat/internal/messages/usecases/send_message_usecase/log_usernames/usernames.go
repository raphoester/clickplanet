// Package log_usernames logs a username the chat could not read. The use case sends the message as a guest's
// and says nothing.
package log_usernames

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase"
)

func New(inner send_message_usecase.Usernames, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  send_message_usecase.Usernames
	logger *slog.Logger
}

var _ send_message_usecase.Usernames = (*Logged)(nil)

// Username logs a failure at Warn: the message still goes out, under the guest's name.
func (l *Logged) Username(ctx context.Context, account messages.AccountID) (string, bool, error) {
	username, found, err := l.inner.Username(ctx, account)
	if err != nil {
		l.logger.Warn("the chat could not read a username; the message is sent as a guest's",
			slog.String("account", account.String()),
			slog.Any("error", err),
		)
	}
	return username, found, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
