package planetv1controller

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
)

func NewErrorInterceptor(logger logging.Logger) connect.Interceptor {
	if logger == nil {
		logger = logging.NewNopLogger()
	}

	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			res, err := next(ctx, req)
			switch {
			case err == nil:
				return res, nil

			case errors.As(err, new(*connect.Error)):
				return nil, err

			case errors.Is(err, domain.ErrInvalidArgument):
				return nil, connect.NewError(connect.CodeInvalidArgument, err)
			}

			logger.Error("rpc failed",
				lf.String("procedure", req.Spec().Procedure),
				lf.Err(err),
			)

			return nil, connect.NewError(connect.CodeInternal, errors.New("internal error"))
		}
	})
}
