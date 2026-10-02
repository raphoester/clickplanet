package send_message_handler

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

// Bare sentinel: the sender must not learn which check tripped.
func toConnect(err error) error {
	if errors.Is(err, messages.ErrInvalidMessage) {
		return connect.NewError(connect.CodeInvalidArgument, messages.ErrInvalidMessage)
	}

	if errors.Is(err, messages.ErrNoAccount) {
		return connect.NewError(connect.CodeUnauthenticated, messages.ErrNoAccount)
	}

	if errors.Is(err, messages.ErrAuthorUnavailable) {
		return connect.NewError(connect.CodeUnavailable, messages.ErrAuthorUnavailable)
	}

	return err
}
