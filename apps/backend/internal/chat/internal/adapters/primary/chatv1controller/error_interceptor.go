package chatv1controller

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/connectutil"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging"
)

func NewErrorInterceptor(logger logging.Logger) connect.Interceptor {
	return connectutil.NewErrorInterceptor(logger, func(err error) *connect.Error {
		if errors.Is(err, domain.ErrInvalidMessage) {
			return connect.NewError(connect.CodeInvalidArgument, domain.ErrInvalidMessage)
		}

		return nil
	})
}
