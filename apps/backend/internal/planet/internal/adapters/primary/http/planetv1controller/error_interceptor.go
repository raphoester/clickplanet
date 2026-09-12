package planetv1controller

import (
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
)

func NewErrorInterceptor(logger *slog.Logger) connect.Interceptor {
	return cpconnect.NewErrorInterceptor(logger, func(err error) *connect.Error {
		if errors.Is(err, domain.ErrInvalidArgument) {
			return connect.NewError(connect.CodeInvalidArgument, err)
		}

		return nil
	})
}
