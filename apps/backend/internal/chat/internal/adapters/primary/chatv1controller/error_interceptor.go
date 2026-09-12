package chatv1controller

import (
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
)

func NewErrorInterceptor(logger *slog.Logger) connect.Interceptor {
	return cpconnect.NewErrorInterceptor(logger, func(err error) *connect.Error {
		if errors.Is(err, domain.ErrInvalidMessage) {
			return connect.NewError(connect.CodeInvalidArgument, domain.ErrInvalidMessage)
		}

		return nil
	})
}
