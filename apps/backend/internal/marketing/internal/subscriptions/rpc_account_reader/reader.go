package rpc_account_reader

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
)

type Dialer interface {
	Dial() (connect.HTTPClient, string, error)
}

const askTimeout = 2 * time.Second

type Reader struct {
	dial Dialer
}

func New(dial Dialer) *Reader {
	return &Reader{dial: dial}
}

func (r *Reader) Account(ctx context.Context, account subscriptions.AccountID) (subscriptions.Account, error) {
	client, baseURL, err := r.dial.Dial()
	if err != nil {
		return subscriptions.Account{}, fmt.Errorf("failed to reach the auth module: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := authv1connect.NewInternalServiceClient(client, baseURL).
		GetAccount(ctx, connect.NewRequest(&authv1.GetAccountRequest{AccountId: account.String()}))
	if err != nil {
		return subscriptions.Account{}, fmt.Errorf("failed to ask auth about the account: %w", err)
	}

	addresses := make([]subscriptions.Address, 0, len(res.Msg.GetEmails()))
	for _, email := range res.Msg.GetEmails() {
		if address, err := subscriptions.AddressOf(email); err == nil {
			addresses = append(addresses, address)
		}
	}
	return subscriptions.Account{ID: account, Linked: res.Msg.GetLinked(), Addresses: addresses}, nil
}
