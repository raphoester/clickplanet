package planetv1controller

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ctxutil"
)

// ClickLimiter decides whether a source may spend another click.
type ClickLimiter interface {
	Allow(key string) bool
}

// NewRateLimitInterceptor throttles Click per source IP, and only Click:
// MapDensity and GetMap are cacheable reads that a proxy in front is expected
// to absorb, and limiting them would punish a page load rather than a bot.
//
// The key is the IP that IPReaderMiddleware put on the context, so a request
// that reached the process without one shares a bucket with every other such
// request — which is the safe direction to fail in.
//
// A refused click answers CodeResourceExhausted, which Connect renders as
// HTTP 429.
func NewRateLimitInterceptor(limiter ClickLimiter) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if req.Spec().Procedure != planetv1connect.ClickServiceClickProcedure {
				return next(ctx, req)
			}

			if !limiter.Allow(ctxutil.GetSourceIP(ctx)) {
				return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("too many clicks"))
			}

			return next(ctx, req)
		}
	})
}
