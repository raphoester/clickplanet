package send_message_handler

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/chatmute"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids"
)

// Bare sentinel: the sender must not learn which check tripped.
func toConnect(err error) error {
	if refused, ok := chatmute.Refusal(err); ok {
		return refused
	}

	if errors.Is(err, messages.ErrInvalidMessage) {
		return connect.NewError(connect.CodeInvalidArgument, messages.ErrInvalidMessage)
	}

	if errors.Is(err, messages.ErrNotShown) {
		return connect.NewError(connect.CodeNotFound, messages.ErrNotShown)
	}

	if errors.Is(err, messages.ErrGuestGif) {
		return connect.NewError(connect.CodePermissionDenied, messages.ErrGuestGif)
	}

	if errors.Is(err, raids.ErrGuest) {
		return connect.NewError(connect.CodePermissionDenied, raids.ErrGuest)
	}

	if errors.Is(err, messages.ErrNoAccount) {
		return connect.NewError(connect.CodeUnauthenticated, messages.ErrNoAccount)
	}

	if errors.Is(err, messages.ErrNotInRoom) {
		return connect.NewError(connect.CodeFailedPrecondition, messages.ErrNotInRoom)
	}

	if errors.Is(err, messages.ErrAuthorUnavailable) {
		return connect.NewError(connect.CodeUnavailable, messages.ErrAuthorUnavailable)
	}

	return err
}
