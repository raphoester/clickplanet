package rpc_gamemaster_battles

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/get_teaser_handler/teaser_query"
)

type Gamemaster interface {
	GetBattle(context.Context, *connect.Request[gamemasterv1.GetBattleRequest]) (*connect.Response[gamemasterv1.GetBattleResponse], error)
}

func New(gamemaster Gamemaster) *Adapter {
	return &Adapter{gamemaster: gamemaster}
}

// What a battle plays, in the gamemaster's words.
type Adapter struct {
	gamemaster Gamemaster
}

func (a *Adapter) Battle(ctx context.Context, battleID string) (teaser_query.Battle, error) {
	res, err := a.gamemaster.GetBattle(ctx, connect.NewRequest(&gamemasterv1.GetBattleRequest{BattleId: battleID}))
	switch {
	case connect.CodeOf(err) == connect.CodeNotFound:
		return teaser_query.Battle{}, fmt.Errorf("%w: %q", teaser_query.ErrNoBattle, battleID)
	case err != nil:
		return teaser_query.Battle{}, fmt.Errorf("failed to ask the gamemaster what the battle %q plays: %w", battleID, err)
	}
	return teaser_query.Battle{Set: res.Msg.GetRuleSet(), Times: res.Msg.GetTimes()}, nil
}
