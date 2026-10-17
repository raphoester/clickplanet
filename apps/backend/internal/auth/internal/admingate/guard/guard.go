package guard

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admingate/admission"
)

type Admitter interface {
	Execute(ctx context.Context, visit admission.Visit) (admission.Operator, error)
}

func New(admitter Admitter) connect.Interceptor {
	return guard{admitter: admitter}
}

type guard struct {
	admitter Admitter
}

func (g guard) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
		admitted, err := g.admitted(ctx, request.Spec().Procedure, request.Header())
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
		admitted, err := g.admitted(ctx, conn.Spec().Procedure, conn.RequestHeader())
		if err != nil {
			return err
		}

		return next(admitted, conn)
	}
}

func (g guard) admitted(ctx context.Context, procedure string, header http.Header) (context.Context, error) {
	operator, err := g.admitter.Execute(ctx, admission.Visit{
		Origin: header.Get("Origin"), Cookie: header.Get("Cookie"), Procedure: procedure,
	})

	switch {
	case err == nil:
		return context.WithValue(ctx, operatorKey{}, operator), nil
	case errors.Is(err, admission.ErrNoSession):
		return nil, connect.NewError(connect.CodeUnauthenticated, admission.ErrNoSession)
	case errors.Is(err, admission.ErrForeignOrigin):
		return nil, connect.NewError(connect.CodePermissionDenied, admission.ErrForeignOrigin)
	case errors.Is(err, admission.ErrNotAnOperator):
		return nil, connect.NewError(connect.CodePermissionDenied, admission.ErrNotAnOperator)
	}

	return nil, fmt.Errorf("failed to admit an operator: %w", err)
}

type operatorKey struct{}

func OperatorOf(ctx context.Context) admission.Operator {
	operator, _ := ctx.Value(operatorKey{}).(admission.Operator)
	return operator
}
