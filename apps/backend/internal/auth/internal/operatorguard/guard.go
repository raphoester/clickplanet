package operatorguard

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/operatorguard/operator_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type Operators interface {
	Operator(ctx context.Context, cookieHeader string) (string, error)
}

func New(operators Operators) connect.Interceptor {
	return guard{operators: operators}
}

type guard struct {
	operators Operators
}

func (g guard) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
		admitted, err := g.admitted(ctx, request.Header())
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
		admitted, err := g.admitted(ctx, conn.RequestHeader())
		if err != nil {
			return err
		}

		return next(admitted, conn)
	}
}

func (g guard) admitted(ctx context.Context, header http.Header) (context.Context, error) {
	account, err := g.operators.Operator(ctx, header.Get("Cookie"))

	switch {
	case err == nil:
		return cpctx.AddOperatorToContext(ctx, account), nil
	case errors.Is(err, operator_query.ErrNoSession):
		return nil, connect.NewError(connect.CodeUnauthenticated, operator_query.ErrNoSession)
	case errors.Is(err, operator_query.ErrNotAnOperator):
		return nil, connect.NewError(connect.CodePermissionDenied, operator_query.ErrNotAnOperator)
	}

	return nil, fmt.Errorf("failed to read the operator: %w", err)
}
