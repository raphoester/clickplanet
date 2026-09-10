package planetv1controller

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/connectutil"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging"
)

func NewErrorInterceptor(logger logging.Logger) connect.Interceptor {
	return connectutil.NewErrorInterceptor(logger, func(err error) *connect.Error {
		if errors.Is(err, domain.ErrInvalidArgument) {
			return connect.NewError(connect.CodeInvalidArgument, err)
		}

		return nil
	})
}
