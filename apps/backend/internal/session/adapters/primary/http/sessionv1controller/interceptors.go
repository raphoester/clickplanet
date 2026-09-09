package sessionv1controller

import (
	"errors"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/session/v1/sessionv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/connectutil"
)

type MintLimiter = connectutil.Limiter

// ErrTooManySessions throttles minting itself. Without it the endpoint is a
// free way to spend this server's siteverify budget, and a way to make a
// blocked address cheap again by collecting tokens from it.
var ErrTooManySessions = errors.New("too many session attempts")

func NewRateLimitInterceptor(limiter MintLimiter) connect.Interceptor {
	return connectutil.NewRateLimitInterceptor(
		limiter,
		ErrTooManySessions,
		sessionv1connect.SessionServiceCreateSessionProcedure,
	)
}
