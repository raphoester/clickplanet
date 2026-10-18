package admin_get_player_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/admin_get_player_handler/admin_player_query"
)

type Query interface {
	Player(ctx context.Context, account players.AccountID) (*playerv1.AdminGetPlayerResponse, error)
}

func New(query Query) AdminGetPlayerHandler {
	return AdminGetPlayerHandler{query: query}
}

type AdminGetPlayerHandler struct {
	query Query
}

func (h AdminGetPlayerHandler) GetPlayer(
	ctx context.Context,
	req *connect.Request[playerv1.AdminGetPlayerRequest],
) (*connect.Response[playerv1.AdminGetPlayerResponse], error) {
	account, err := players.AccountIDOf(req.Msg.GetAccountId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	player, err := h.query.Player(ctx, account)
	switch {
	case errors.Is(err, admin_player_query.ErrNoPlayer):
		return nil, connect.NewError(connect.CodeNotFound, admin_player_query.ErrNoPlayer)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	res := connect.NewResponse(player)
	res.Header().Set("Cache-Control", "no-store")
	return res, nil
}
