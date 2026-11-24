package rpc_accounts

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

type Player interface {
	GetAccounts(ctx context.Context, req *connect.Request[playerv1.GetAccountsRequest]) (*connect.Response[playerv1.GetAccountsResponse], error)
}

const askTimeout = 10 * time.Second

func New(player Player) *Accounts {
	return &Accounts{player: player}
}

type Accounts struct {
	player Player
}

// The accounts that signed in with a provider: the others are guests, or no longer known.
func (a *Accounts) Linked(ctx context.Context, accounts []notes.AccountID) (*cpcolls.Set[notes.AccountID], error) {
	linked := cpcolls.NewSet[notes.AccountID]()
	if len(accounts) == 0 {
		return linked, nil
	}
	ids := make([]string, 0, len(accounts))
	for _, account := range accounts {
		ids = append(ids, account.String())
	}

	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()
	res, err := a.player.GetAccounts(ctx, connect.NewRequest(&playerv1.GetAccountsRequest{AccountIds: ids}))
	if err != nil {
		return nil, fmt.Errorf("failed to call player.v1.InternalService/GetAccounts: %w", err)
	}

	for _, account := range res.Msg.GetAccounts() {
		if !account.GetLinked() {
			continue
		}
		id, err := notes.AccountIDOf(account.GetAccountId())
		if err != nil {
			return nil, fmt.Errorf("player answered an account that is not one: %w", err)
		}
		linked.Add(id)
	}
	return linked, nil
}
