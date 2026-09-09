package planetv1controller

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/connectutil"
)

type ClickLimiter = connectutil.Limiter

func NewRateLimitInterceptor(limiter ClickLimiter) connect.Interceptor {
	return connectutil.NewRateLimitInterceptor(
		limiter,
		errors.New("too many clicks"),
		planetv1connect.ClickServiceClickProcedure,
	)
}
