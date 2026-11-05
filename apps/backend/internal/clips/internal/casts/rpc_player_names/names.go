package rpc_player_names

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
)

type Player interface {
	GetPlayers(
		ctx context.Context,
		req *connect.Request[playerv1.GetPlayersRequest],
	) (*connect.Response[playerv1.GetPlayersResponse], error)
}

const askTimeout = 5 * time.Second

func New(player Player) *Names {
	return &Names{player: player}
}

type Names struct {
	player Player
}

var _ casts.Usernames = (*Names)(nil)

func (n *Names) Of(ctx context.Context, accounts []casts.AccountID) (map[casts.AccountID]string, error) {
	names := make(map[casts.AccountID]string, len(accounts))
	if len(accounts) == 0 {
		return names, nil
	}

	ids := make([]string, 0, len(accounts))
	for _, account := range accounts {
		ids = append(ids, account.String())
	}

	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	answer, err := n.player.GetPlayers(ctx, connect.NewRequest(&playerv1.GetPlayersRequest{AccountIds: ids}))
	if err != nil {
		return nil, fmt.Errorf("failed to ask the player module who these accounts are: %w", err)
	}

	for _, player := range answer.Msg.GetPlayers() {
		if player.GetGuest() {
			continue
		}
		account, err := casts.AccountIDOf(player.GetAccountId())
		if err != nil {
			return nil, fmt.Errorf("the player module answered for %q: %w", player.GetAccountId(), err)
		}
		names[account] = player.GetName()
	}
	return names, nil
}
