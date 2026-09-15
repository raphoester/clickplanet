package cpbootstrap

import (
	"context"

	"connectrpc.com/connect"
)

// drainInterceptor ends every open stream once shutdown starts.
//
// http.Server.Shutdown waits for every connection to go idle, and a live stream
// never does: without this, each deploy waited the whole ShutdownTimeout and then
// cut the streams hard, which the proxy answered with a 502. A stream handler
// already returns when its context is cancelled, so cancelling that context is
// all it takes for the stream to end cleanly, with an end-of-stream message.
//
// It cannot be done with http.Server.BaseContext: that context is every
// request's parent, so cancelling it would also cancel the unary calls that
// Shutdown is waiting on to finish normally.
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

// A stream opened after shutdown started is ended at once: AfterFunc runs
// straight away on a context that is already done.
func (d drainInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		stop := context.AfterFunc(d.draining, cancel)
		defer stop()

		return next(ctx, conn)
	}
}
