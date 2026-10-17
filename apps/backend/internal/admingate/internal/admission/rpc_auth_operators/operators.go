package rpc_auth_operators

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/admingate/internal/admission"
)

const askTimeout = 2 * time.Second

type Auth interface {
	GetOperator(ctx context.Context, req *connect.Request[authv1.GetOperatorRequest]) (*connect.Response[authv1.GetOperatorResponse], error)
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
