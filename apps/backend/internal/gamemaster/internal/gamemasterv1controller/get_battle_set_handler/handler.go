package get_battle_set_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_battle_set_handler/battle_set_query"
)

type Query interface {
	RuleSet(ctx context.Context, battleID string) (string, error)
}

func New(query Query) GetBattleSetHandler {
	return GetBattleSetHandler{query: query}
}

type GetBattleSetHandler struct {
	query Query
}

func (h GetBattleSetHandler) GetBattleSet(
	ctx context.Context,
	req *connect.Request[gamemasterv1.GetBattleSetRequest],
) (*connect.Response[gamemasterv1.GetBattleSetResponse], error) {
	set, err := h.query.RuleSet(ctx, req.Msg.GetBattleId())
	switch {
	case errors.Is(err, battle_set_query.ErrNoBattle):
		return nil, connect.NewError(connect.CodeNotFound, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&gamemasterv1.GetBattleSetResponse{RuleSet: set}), nil
}
