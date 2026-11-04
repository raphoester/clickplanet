package rpc_player_accounts

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
)

type Player interface {
	GetAccounts(
		ctx context.Context,
		req *connect.Request[playerv1.GetAccountsRequest],
	) (*connect.Response[playerv1.GetAccountsResponse], error)
}

const askTimeout = 2 * time.Second

func New(player Player) *Reader {
	return &Reader{player: player}
}

type Reader struct {
	player Player
}

func (r *Reader) Account(ctx context.Context, account accounts.AccountID) (accounts.Account, error) {
	known, err := r.Accounts(ctx, []accounts.AccountID{account})
	if err != nil {
		return accounts.Account{}, err
	}
	return known[account], nil
}

func (r *Reader) Accounts(ctx context.Context, asked []accounts.AccountID) (map[accounts.AccountID]accounts.Account, error) {
	found := make(map[accounts.AccountID]accounts.Account, len(asked))
	if len(asked) == 0 {
		return found, nil
	}
	ids := make([]string, len(asked))
	for i, account := range asked {
		ids[i] = account.String()
	}

	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()
	res, err := r.player.GetAccounts(ctx, connect.NewRequest(&playerv1.GetAccountsRequest{AccountIds: ids}))
	if err != nil {
		return nil, fmt.Errorf("failed to call player.v1.InternalService/GetAccounts: %w", err)
	}

	for _, answered := range res.Msg.GetAccounts() {
		account, err := accounts.AccountIDOf(answered.GetAccountId())
		if err != nil {
			return nil, fmt.Errorf("player answered %w", err)
		}
		found[account] = accountOf(answered.GetLinked(), answered.GetCreatedAtUnixMs())
	}
	return found, nil
}

func accountOf(linked bool, createdAtUnixMs int64) accounts.Account {
	if createdAtUnixMs == 0 {
		return accounts.AccountOf(linked, time.Time{})
	}
	return accounts.AccountOf(linked, time.UnixMilli(createdAtUnixMs).UTC())
}
