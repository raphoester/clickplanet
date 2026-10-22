package rpc_token_holders

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/mcp/internal/tools"
)

const askTimeout = 2 * time.Second

type Auth interface {
	GetTokenOperator(ctx context.Context, req *connect.Request[authv1.GetTokenOperatorRequest]) (*connect.Response[authv1.GetTokenOperatorResponse], error)
}

func New(auth Auth) *Holders {
	return &Holders{auth: auth}
}

// Who holds an access token, from auth.v1.InternalService/GetTokenOperator: the operator, read again on every call.
type Holders struct {
	auth Auth
}

func (h *Holders) HolderError(ctx context.Context, token string) error {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	_, err := h.auth.GetTokenOperator(ctx, connect.NewRequest(&authv1.GetTokenOperatorRequest{Token: token}))
	switch {
	case err == nil:
		return nil
	case connect.CodeOf(err) == connect.CodeUnauthenticated:
		return fmt.Errorf("%w: %w", tools.ErrNoToken, err)
	case connect.CodeOf(err) == connect.CodePermissionDenied:
		return fmt.Errorf("%w: %w", tools.ErrNotAnOperator, err)
	}
	return fmt.Errorf("failed to ask auth who holds the token: %w", err)
}
