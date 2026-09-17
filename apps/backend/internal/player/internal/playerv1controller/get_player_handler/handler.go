// Package get_player_handler serves player.v1.PlayerService/GetPlayer.
package get_player_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
)

// maxAge lets a proxy serve one answer to everyone who opens the same player within it. The stats move with
// every take, and nobody reads them to the tile.
const maxAge = 10

type UseCase interface {
	Execute(ctx context.Context, name string) (players.Player, error)
}

func New(useCase UseCase) GetPlayerHandler {
	return GetPlayerHandler{useCase: useCase}
}

type GetPlayerHandler struct {
	useCase UseCase
}

// GetPlayer needs no token: the session interceptor does not list it.
func (h GetPlayerHandler) GetPlayer(
	ctx context.Context,
	req *connect.Request[playerv1.GetPlayerRequest],
) (*connect.Response[playerv1.GetPlayerResponse], error) {
	player, err := h.useCase.Execute(ctx, req.Msg.GetName())
	switch {
	case errors.Is(err, players.ErrNoProfile):
		return nil, connect.NewError(connect.CodeNotFound, players.ErrNoProfile)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	res := connect.NewResponse(&playerv1.GetPlayerResponse{Player: playermessage.Player(player)})
	res.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
	return res, nil
}
