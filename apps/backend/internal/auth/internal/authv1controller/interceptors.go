package authv1controller

import (
	"errors"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
)

type MintLimiter = cpconnect.Limiter

// ErrTooManySessions throttles minting and signing in: each costs a round trip to a third party and may create an account.
var ErrTooManySessions = errors.New("too many session attempts")

func NewRateLimitInterceptor(limiter MintLimiter) connect.Interceptor {
	return cpconnect.NewRateLimitInterceptor(limiter, ErrTooManySessions,
		authv1connect.AuthServiceCreateSessionProcedure,
		authv1connect.AuthServiceStartSignInProcedure,
		authv1connect.AuthServiceCompleteSignInProcedure,
	)
}
