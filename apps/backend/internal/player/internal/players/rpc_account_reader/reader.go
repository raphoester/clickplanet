// Package rpc_account_reader asks the auth module about an account, over the internal listener.
package rpc_account_reader

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

// Dialer is cpbootstrap's internal listener: the only way one module reaches another.
type Dialer interface {
	Dial() (connect.HTTPClient, string, error)
}

// askTimeout bounds one call. It is loopback, so this is a stuck socket rather than a slow network.
const askTimeout = 2 * time.Second

type Reader struct {
	dial Dialer
}

func New(dial Dialer) *Reader {
	return &Reader{dial: dial}
}

// Linked asks auth on every call: a player chooses a name rarely, and an account links at any time.
func (r *Reader) Linked(ctx context.Context, account players.AccountID) (bool, error) {
	client, baseURL, err := r.dial.Dial()
	if err != nil {
		return false, fmt.Errorf("failed to reach the auth module: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := authv1connect.NewInternalServiceClient(client, baseURL).
		GetAccount(ctx, connect.NewRequest(&authv1.GetAccountRequest{AccountId: account.String()}))
	if err != nil {
		return false, fmt.Errorf("failed to ask auth whether the account is linked: %w", err)
	}

	return res.Msg.GetLinked(), nil
}
