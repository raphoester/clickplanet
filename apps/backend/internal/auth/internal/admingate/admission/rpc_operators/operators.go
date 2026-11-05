package rpc_operators

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admingate/admission"
)

const askTimeout = 2 * time.Second

type Auth interface {
	GetOperator(ctx context.Context, req *connect.Request[authv1.GetOperatorRequest]) (*connect.Response[authv1.GetOperatorResponse], error)
	GetTokenOperator(ctx context.Context, req *connect.Request[authv1.GetTokenOperatorRequest]) (*connect.Response[authv1.GetTokenOperatorResponse], error)
}

func New(auth Auth) *Operators {
	return &Operators{auth: auth}
}

type Operators struct {
	auth Auth
}

func (o *Operators) Operator(ctx context.Context, cookie string) (admission.Operator, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := o.auth.GetOperator(ctx, connect.NewRequest(&authv1.GetOperatorRequest{Cookie: cookie}))
	if err == nil {
		return admission.Operator(res.Msg.GetAccountId()), nil
	}
	if connect.CodeOf(err) == connect.CodeUnauthenticated {
		return "", fmt.Errorf("%w: %w", admission.ErrNoSession, err)
	}
	if connect.CodeOf(err) == connect.CodePermissionDenied {
		return "", fmt.Errorf("%w: %w", admission.ErrNotAnOperator, err)
	}

	return "", fmt.Errorf("failed to ask auth who operates: %w", err)
}

func (o *Operators) Connection(ctx context.Context, token string) (admission.Caller, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := o.auth.GetTokenOperator(ctx, connect.NewRequest(&authv1.GetTokenOperatorRequest{Token: token}))
	if err == nil {
		return admission.ConnectionOf(admission.Operator(res.Msg.GetAccountId()), res.Msg.GetClient()), nil
	}
	if connect.CodeOf(err) == connect.CodeUnauthenticated {
		return admission.Caller{}, fmt.Errorf("%w: %w", admission.ErrNoToken, err)
	}
	if connect.CodeOf(err) == connect.CodePermissionDenied {
		return admission.Caller{}, fmt.Errorf("%w: %w", admission.ErrNotAnOperator, err)
	}

	return admission.Caller{}, fmt.Errorf("failed to ask auth whose token it is: %w", err)
}
