package rpc_addresses

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

type Player interface {
	GetEmail(ctx context.Context, req *connect.Request[playerv1.GetEmailRequest]) (*connect.Response[playerv1.GetEmailResponse], error)
}

const askTimeout = 5 * time.Second

func New(player Player) *Addresses {
	return &Addresses{player: player}
}

type Addresses struct {
	player Player
}

var _ notes.AddressBook = (*Addresses)(nil)

func (a *Addresses) Address(ctx context.Context, account notes.AccountID) (cpmail.Address, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := a.player.GetEmail(ctx, connect.NewRequest(&playerv1.GetEmailRequest{AccountId: account.String()}))
	if err != nil {
		return "", fmt.Errorf("failed to call player.v1.InternalService/GetEmail: %w", err)
	}
	if res.Msg.Email == nil {
		return "", fmt.Errorf("%w: the account holds none", notes.ErrNoReplyAddress)
	}

	address, err := cpmail.AddressOf(res.Msg.GetEmail())
	if err != nil {
		return "", fmt.Errorf("%w: the account's: %w", notes.ErrNoReplyAddress, err)
	}
	return address, nil
}

// Each account's address, the accounts that hold none left out.
func (a *Addresses) Addresses(ctx context.Context, accounts []notes.AccountID) (map[notes.AccountID]cpmail.Address, error) {
	held := make(map[notes.AccountID]cpmail.Address, len(accounts))
	for _, account := range accounts {
		address, err := a.Address(ctx, account)
		if errors.Is(err, notes.ErrNoReplyAddress) {
			continue
		}
		if err != nil {
			return nil, err
		}
		held[account] = address
	}
	return held, nil
}
