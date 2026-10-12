package opsv1controller

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/ops/internal/access"
)

const assertionHeader = "Cf-Access-Jwt-Assertion"

var ErrNoAccess = errors.New("this service answers a caller Cloudflare Access let in, and this request carries no valid assertion")

type Assertions interface {
	Caller(ctx context.Context, assertion string) (access.Caller, error)
}

func NewAccessInterceptor(assertions Assertions) connect.Interceptor {
	return accessInterceptor{assertions: assertions}
}

type accessInterceptor struct {
	assertions Assertions
}

func (i accessInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		admitted, err := i.admitted(ctx, req.Header())
		if err != nil {
			return nil, err
		}

		return next(admitted, req)
	}
}

func (i accessInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

// A stream is checked too: a procedure added later must not be the one nobody guards.
func (i accessInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		admitted, err := i.admitted(ctx, conn.RequestHeader())
		if err != nil {
			return err
		}

		return next(admitted, conn)
	}
}

func (i accessInterceptor) admitted(ctx context.Context, header http.Header) (context.Context, error) {
	caller, err := i.assertions.Caller(ctx, header.Get(assertionHeader))
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, ErrNoAccess)
	}

	return access.WithCaller(ctx, caller), nil
}
