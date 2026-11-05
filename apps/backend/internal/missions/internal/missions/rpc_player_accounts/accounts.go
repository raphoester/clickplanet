package rpc_player_accounts

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
)

type Player interface {
	GetAccounts(
		ctx context.Context,
		req *connect.Request[playerv1.GetAccountsRequest],
	) (*connect.Response[playerv1.GetAccountsResponse], error)
}

const askTimeout = 2 * time.Second

func New(player Player) Accounts {
	return Accounts{player: player}
}

type Accounts struct {
	player Player
}

// An account player does not answer is not signed in, as auth answers one it does not know.
func (a Accounts) Linked(ctx context.Context, account missions.AccountID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := a.player.GetAccounts(ctx, connect.NewRequest(&playerv1.GetAccountsRequest{AccountIds: []string{account.String()}}))
	if err != nil {
		return false, fmt.Errorf("failed to ask player whether %s is signed in: %w", account, err)
	}
	for _, answered := range res.Msg.GetAccounts() {
		if answered.GetAccountId() == account.String() {
			return answered.GetLinked(), nil
		}
	}
	return false, nil
}
