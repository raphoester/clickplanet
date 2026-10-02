package cpbootstrap

import (
	"context"

	"connectrpc.com/connect"
)

// Not http.Server.BaseContext: cancelling that would also cut unary calls Shutdown awaits.
type drainInterceptor struct {
	draining context.Context //nolint:containedctx // a signal shared by every stream, not a request's context.
}

func newDrainInterceptor(draining context.Context) drainInterceptor {
	return drainInterceptor{draining: draining}
}

func (d drainInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return next
}

func (d drainInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (d drainInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		stop := context.AfterFunc(d.draining, cancel)
		defer stop()

		return next(ctx, conn)
	}
}
