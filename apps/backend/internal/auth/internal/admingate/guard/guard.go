package guard

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admingate/admission"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
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
		return cpctx.AddOperatorToContext(context.WithValue(ctx, callerKey{}, caller), operatorOf(caller)), nil
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

// RFC 9110: the scheme is case-insensitive; another scheme is no token.
func bearerOf(authorization string) string {
	scheme, token, found := strings.Cut(authorization, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}

// What the modules behind the gate read of who calls, to keep it with what the call did (a mute, a rename). A sign-in
// call is admitted before anybody is known, and has no account.
func operatorOf(caller admission.Caller) cpctx.Operator {
	account, err := uuid.Parse(string(caller.Operator()))
	if err != nil {
		return cpctx.OperatorOf(cpsession.NoAccount, caller.Client())
	}
	return cpctx.OperatorOf(cpsession.AccountID(account), caller.Client())
}

type callerKey struct{}

func CallerOf(ctx context.Context) admission.Caller {
	caller, _ := ctx.Value(callerKey{}).(admission.Caller)
	return caller
}
