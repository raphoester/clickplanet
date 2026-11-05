package get_battle_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_battle_handler/battle_query"
)

type Query interface {
	Battle(ctx context.Context, battleID string) (*gamemasterv1.GetBattleResponse, error)
}

func New(query Query) GetBattleHandler {
	return GetBattleHandler{query: query}
}

type GetBattleHandler struct {
	query Query
}

func (h GetBattleHandler) GetBattle(
	ctx context.Context,
	req *connect.Request[gamemasterv1.GetBattleRequest],
) (*connect.Response[gamemasterv1.GetBattleResponse], error) {
	battle, err := h.query.Battle(ctx, req.Msg.GetBattleId())
	switch {
	case errors.Is(err, battle_query.ErrNoBattle):
		return nil, connect.NewError(connect.CodeNotFound, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(battle), nil
}
