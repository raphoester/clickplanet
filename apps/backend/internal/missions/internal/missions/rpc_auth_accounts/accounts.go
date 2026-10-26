package rpc_auth_accounts

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
)

type Auth interface {
	GetAccount(ctx context.Context, req *connect.Request[authv1.GetAccountRequest]) (*connect.Response[authv1.GetAccountResponse], error)
}

const askTimeout = 2 * time.Second

func New(auth Auth) Accounts {
	return Accounts{auth: auth}
}

type Accounts struct {
	auth Auth
}

func (a Accounts) Linked(ctx context.Context, account missions.AccountID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := a.auth.GetAccount(ctx, connect.NewRequest(&authv1.GetAccountRequest{AccountId: account.String()}))
	if err != nil {
		return false, fmt.Errorf("failed to ask auth whether %s is signed in: %w", account, err)
	}
	return res.Msg.GetLinked(), nil
}
