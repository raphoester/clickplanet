package planetv1controller

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cplogging"
)

func NewErrorInterceptor(logger cplogging.Logger) connect.Interceptor {
	return cpconnect.NewErrorInterceptor(logger, func(err error) *connect.Error {
		if errors.Is(err, domain.ErrInvalidArgument) {
			return connect.NewError(connect.CodeInvalidArgument, err)
		}

		return nil
	})
}
