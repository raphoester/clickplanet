package rpc_gamemaster_battles

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/get_teaser_handler/teaser_query"
)

type Gamemaster interface {
	GetBattleSet(context.Context, *connect.Request[gamemasterv1.GetBattleSetRequest]) (*connect.Response[gamemasterv1.GetBattleSetResponse], error)
}

func New(gamemaster Gamemaster) *Adapter {
	return &Adapter{gamemaster: gamemaster}
}

// The set a battle plays, in the gamemaster's words.
type Adapter struct {
	gamemaster Gamemaster
}

func (a *Adapter) SetOf(ctx context.Context, battleID string) (string, error) {
	res, err := a.gamemaster.GetBattleSet(ctx, connect.NewRequest(&gamemasterv1.GetBattleSetRequest{BattleId: battleID}))
	switch {
	case connect.CodeOf(err) == connect.CodeNotFound:
		return "", fmt.Errorf("%w: %q", teaser_query.ErrNoBattle, battleID)
	case err != nil:
		return "", fmt.Errorf("failed to ask the gamemaster which set the battle %q plays: %w", battleID, err)
	}
	return res.Msg.GetRuleSet(), nil
}
