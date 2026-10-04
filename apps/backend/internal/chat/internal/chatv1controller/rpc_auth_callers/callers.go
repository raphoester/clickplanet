package rpc_auth_callers

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type Dialer interface {
	Dial() (connect.HTTPClient, string, error)
}

const askTimeout = time.Second

func New(dial Dialer) *Callers {
	return &Callers{dial: dial}
}

type Callers struct {
	dial Dialer
}

func (c *Callers) Caller(ctx context.Context, cookie string) (messages.AccountID, error) {
	client, baseURL, err := c.dial.Dial()
	if err != nil {
		return messages.NoAccount, fmt.Errorf("failed to reach the auth module: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := authv1connect.NewInternalServiceClient(client, baseURL).GetCaller(ctx,
		connect.NewRequest(&authv1.GetCallerRequest{Cookie: cookie}))
	if err != nil {
		return messages.NoAccount, fmt.Errorf("failed to ask the auth module whose cookie this is: %w", err)
	}

	return messages.AccountIDOf(res.Msg.GetAccountId()), nil
}
