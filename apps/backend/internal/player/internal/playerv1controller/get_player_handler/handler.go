package get_player_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_player_handler/player_query"
)

const maxAge = 10

type Query interface {
	Player(ctx context.Context, name string) (*playerv1.GetPlayerResponse, error)
}

func New(query Query) GetPlayerHandler {
	return GetPlayerHandler{query: query}
}

type GetPlayerHandler struct {
	query Query
}

func (h GetPlayerHandler) GetPlayer(
	ctx context.Context,
	req *connect.Request[playerv1.GetPlayerRequest],
) (*connect.Response[playerv1.GetPlayerResponse], error) {
	player, err := h.query.Player(ctx, req.Msg.GetName())
	switch {
	case errors.Is(err, player_query.ErrNoPlayer):
		return nil, connect.NewError(connect.CodeNotFound, player_query.ErrNoPlayer)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	res := connect.NewResponse(player)
	res.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
	return res, nil
}
