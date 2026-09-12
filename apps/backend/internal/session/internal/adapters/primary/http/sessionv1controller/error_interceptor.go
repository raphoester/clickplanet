package sessionv1controller

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/session/internal/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/logging"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/logging/lf"
)

// ErrRefused is the whole of what a refused caller is told. The reason — a
// forged token, a token for another site, siteverify being unreachable — is
// logged here and goes no further.
var ErrRefused = errors.New("could not start a session")

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

			case errors.Is(err, domain.ErrAttestationFailed):
				logger.Info("refused a session",
					lf.String("procedure", req.Spec().Procedure),
					lf.Err(err),
				)
				return nil, connect.NewError(connect.CodePermissionDenied, ErrRefused)
			}

			logger.Error("rpc failed",
				lf.String("procedure", req.Spec().Procedure),
				lf.Err(err),
			)

			return nil, connect.NewError(connect.CodeInternal, errors.New("internal error"))
		}
	})
}
