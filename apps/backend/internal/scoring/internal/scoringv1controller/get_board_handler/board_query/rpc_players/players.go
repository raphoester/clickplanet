package rpc_players

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_board_handler/board_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings"
)

type Player interface {
	GetPlayers(
		ctx context.Context,
		req *connect.Request[playerv1.GetPlayersRequest],
	) (*connect.Response[playerv1.GetPlayersResponse], error)
}

const askTimeout = time.Second

func New(player Player) *Players {
	return &Players{player: player}
}

type Players struct {
	player Player
}

var _ board_query.Players = (*Players)(nil)

func (a *Players) Players(
	ctx context.Context,
	accounts []standings.AccountID,
) (map[standings.AccountID]*playerv1.Player, error) {
	found := make(map[standings.AccountID]*playerv1.Player, len(accounts))
	if len(accounts) == 0 {
		return found, nil
	}

	ids := make([]string, 0, len(accounts))
	for _, account := range accounts {
		ids = append(ids, account.String())
	}

	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := a.player.GetPlayers(ctx, connect.NewRequest(&playerv1.GetPlayersRequest{AccountIds: ids}))
	if err != nil {
		return nil, fmt.Errorf("failed to ask the player module who these accounts are: %w", err)
	}

	for _, player := range res.Msg.GetPlayers() {
		id, err := uuid.Parse(player.GetAccountId())
		if err != nil {
			return nil, fmt.Errorf("the player module answered for %q, which is not an account: %w", player.GetAccountId(), err)
		}
		found[standings.AccountID(id)] = player
	}
	return found, nil
}
