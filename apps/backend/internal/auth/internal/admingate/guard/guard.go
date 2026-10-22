package guard

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admingate/admission"
)

type Admitter interface {
	Execute(ctx context.Context, visit admission.Visit) (admission.Caller, error)
}

func New(admitter Admitter) connect.Interceptor {
	return guard{admitter: admitter}
}

type guard struct {
	admitter Admitter
}

func (g guard) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
		admitted, err := g.admitted(ctx, request.Spec(), request.Header())
		if err != nil {
			return nil, err
		}

		return next(admitted, request)
	}
}

func (g guard) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (g guard) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		admitted, err := g.admitted(ctx, conn.Spec(), conn.RequestHeader())
		if err != nil {
			return err
		}

		return next(admitted, conn)
	}
}

func (g guard) admitted(ctx context.Context, spec connect.Spec, header http.Header) (context.Context, error) {
	caller, err := g.admitter.Execute(ctx, admission.Visit{
		Origin:    header.Get("Origin"),
		Cookie:    header.Get("Cookie"),
		Token:     bearerOf(header.Get("Authorization")),
		Procedure: spec.Procedure,
	})

	switch {
	case err == nil:
		return context.WithValue(ctx, callerKey{}, caller), nil
	case errors.Is(err, admission.ErrNoSession):
		return nil, connect.NewError(connect.CodeUnauthenticated, admission.ErrNoSession)
	case errors.Is(err, admission.ErrNoToken):
		return nil, connect.NewError(connect.CodeUnauthenticated, admission.ErrNoToken)
	case errors.Is(err, admission.ErrForeignOrigin):
		return nil, connect.NewError(connect.CodePermissionDenied, admission.ErrForeignOrigin)
	case errors.Is(err, admission.ErrNotAnOperator):
		return nil, connect.NewError(connect.CodePermissionDenied, admission.ErrNotAnOperator)
	case errors.Is(err, admission.ErrSessionOnly):
		return nil, connect.NewError(connect.CodePermissionDenied, admission.ErrSessionOnly)
	}

	return nil, fmt.Errorf("failed to admit an operator: %w", err)
}

// The scheme is case-insensitive (RFC 9110); any other scheme is no token, and the call is judged by its cookie.
func bearerOf(authorization string) string {
	scheme, token, found := strings.Cut(authorization, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}

type callerKey struct{}

func CallerOf(ctx context.Context) admission.Caller {
	caller, _ := ctx.Value(callerKey{}).(admission.Caller)
	return caller
}
