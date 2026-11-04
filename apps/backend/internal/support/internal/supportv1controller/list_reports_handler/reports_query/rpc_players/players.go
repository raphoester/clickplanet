package rpc_players

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/list_reports_handler/reports_query"
)

type Player interface {
	GetPlayers(
		ctx context.Context,
		req *connect.Request[playerv1.GetPlayersRequest],
	) (*connect.Response[playerv1.GetPlayersResponse], error)
}

const askTimeout = 2 * time.Second

func New(player Player) *Players {
	return &Players{player: player}
}

type Players struct {
	player Player
}

var _ reports_query.Players = (*Players)(nil)

func (a *Players) Players(
	ctx context.Context,
	accounts []cpsession.AccountID,
) (map[cpsession.AccountID]*playerv1.Player, error) {
	found := make(map[cpsession.AccountID]*playerv1.Player, len(accounts))
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
		found[cpsession.AccountID(id)] = player
	}
	return found, nil
}
