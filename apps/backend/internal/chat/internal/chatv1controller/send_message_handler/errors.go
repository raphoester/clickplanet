package send_message_handler

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

// toConnect sends the bare sentinel: a sender learns they were refused, not which
// check tripped. Anything else is left for the error net.
func toConnect(err error) error {
	if errors.Is(err, messages.ErrInvalidMessage) {
		return connect.NewError(connect.CodeInvalidArgument, messages.ErrInvalidMessage)
	}

	return err
}
