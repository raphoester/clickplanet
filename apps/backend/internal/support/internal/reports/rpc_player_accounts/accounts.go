package rpc_player_accounts

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
)

type Player interface {
	FindAccount(
		ctx context.Context,
		req *connect.Request[playerv1.FindAccountRequest],
	) (*connect.Response[playerv1.FindAccountResponse], error)
}

const askTimeout = 2 * time.Second

func New(player Player) *Accounts {
	return &Accounts{player: player}
}

type Accounts struct {
	player Player
}

func (a *Accounts) AccountOf(ctx context.Context, name reports.ShownName) (reports.AccountID, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := a.player.FindAccount(ctx, connect.NewRequest(&playerv1.FindAccountRequest{Name: string(name)}))
	if connect.CodeOf(err) == connect.CodeNotFound {
		return reports.AccountID{}, fmt.Errorf("%w: %q", reports.ErrNoPlayer, string(name))
	}
	if err != nil {
		return reports.AccountID{}, fmt.Errorf("failed to ask the player module whose name this is: %w", err)
	}

	account, err := notes.AccountIDOf(res.Msg.GetAccountId())
	if err != nil {
		return reports.AccountID{}, fmt.Errorf("the player module answered what is not an account: %w", err)
	}
	return account, nil
}
