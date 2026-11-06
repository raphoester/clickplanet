package rpc_player_names

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
)

type Player interface {
	GetPlayers(
		ctx context.Context,
		req *connect.Request[playerv1.GetPlayersRequest],
	) (*connect.Response[playerv1.GetPlayersResponse], error)
}

const askTimeout = 5 * time.Second

func New(player Player) Names {
	return Names{player: player}
}

type Names struct {
	player Player
}

var _ links.Names = Names{}

// No name for a guest, which has its code alone, and for an account player does not show: one its player asked to
// delete, or one it never named.
func (n Names) NameOf(ctx context.Context, account links.AccountID) (links.Name, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := n.player.GetPlayers(ctx, connect.NewRequest(&playerv1.GetPlayersRequest{AccountIds: []string{account.String()}}))
	if err != nil {
		return "", fmt.Errorf("failed to ask player the name of %s: %w", account, err)
	}
	for _, player := range res.Msg.GetPlayers() {
		if player.GetAccountId() == account.String() && !player.GetGuest() {
			return links.Name(player.GetName()), nil
		}
	}
	return "", nil
}
