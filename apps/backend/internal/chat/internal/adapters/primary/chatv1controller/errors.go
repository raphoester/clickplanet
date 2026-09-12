package chatv1controller

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/domain"
)

// toConnect leaves anything it does not recognise alone, for the error net
// cpbootstrap wraps every service in to log once and answer "internal error".
//
// The sentinel's own sentence never travels: a sender learns that they were
// refused, not which check tripped.
func toConnect(err error) error {
	if errors.Is(err, domain.ErrInvalidMessage) {
		return connect.NewError(connect.CodeInvalidArgument, domain.ErrInvalidMessage)
	}

	return err
}
