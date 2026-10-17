package cpbootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

var errForeignOrigin = errors.New("the operator services answer the admin page only")

// A browser always says where a call comes from, so a page of another origin cannot ride the cookie it holds.
func newOriginGuard(origin string) connect.Interceptor {
	return originGuard{origin: origin}
}

type originGuard struct {
	origin string
}

func (g originGuard) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
		if err := g.refusal(request.Header()); err != nil {
			return nil, err
		}
		return next(ctx, request)
	}
}

func (g originGuard) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (g originGuard) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		if err := g.refusal(conn.RequestHeader()); err != nil {
			return err
		}
		return next(ctx, conn)
	}
}

func (g originGuard) refusal(header http.Header) error {
	if header.Get("Origin") != g.origin {
		return connect.NewError(connect.CodePermissionDenied, errForeignOrigin)
	}
	return nil
}

// Inside the module's guard, so every line names its operator: reads included, which no audit line covers.
func newCallLog(logger *slog.Logger) connect.Interceptor {
	return callLog{logger: logger}
}

type callLog struct {
	logger *slog.Logger
}

func (c callLog) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
		response, err := next(ctx, request)
		c.log(ctx, request.Spec().Procedure, err)
		return response, err
	}
}

func (c callLog) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (c callLog) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		err := next(ctx, conn)
		c.log(ctx, conn.Spec().Procedure, err)
		return err
	}
}

func (c callLog) log(ctx context.Context, procedure string, err error) {
	code := "ok"
	if err != nil {
		code = connect.CodeOf(err).String()
	}

	c.logger.Info("operator call",
		slog.String("operator", cpctx.GetOperator(ctx)),
		slog.String("procedure", procedure),
		slog.String("code", code),
	)
}
