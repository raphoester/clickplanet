package rpc_planet_recounts

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Planet interface {
	GetBombsAndFortifies(
		ctx context.Context,
		req *connect.Request[planetv1.GetBombsAndFortifiesRequest],
	) (*connect.Response[planetv1.GetBombsAndFortifiesResponse], error)
}

const askTimeout = 30 * time.Second

type Reader struct {
	planet Planet
}

func New(planet Planet) *Reader {
	return &Reader{planet: planet}
}

func (r *Reader) Recounts(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]players.Recount, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	ids := make([]string, len(accounts))
	for i, account := range accounts {
		ids[i] = account.String()
	}
	res, err := r.planet.GetBombsAndFortifies(ctx, connect.NewRequest(&planetv1.GetBombsAndFortifiesRequest{AccountIds: ids}))
	if err != nil {
		return nil, fmt.Errorf("failed to ask planet for the accounts' bombs and fortifies: %w", err)
	}

	recounts := make(map[players.AccountID]players.Recount, len(res.Msg.GetAccounts()))
	for _, kept := range res.Msg.GetAccounts() {
		account, err := players.AccountIDOf(kept.GetAccountId())
		if err != nil {
			return nil, fmt.Errorf("planet answered for an id that is not an account: %w", err)
		}
		recounts[account] = players.Recount{BombsDropped: kept.GetBombsDropped(), TilesFortified: kept.GetTilesFortified()}
	}
	return recounts, nil
}
