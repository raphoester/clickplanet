package get_player_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
)

const maxAge = 10

type UseCase interface {
	Execute(ctx context.Context, name string) (get_player_usecase.Player, error)
}

func New(useCase UseCase) GetPlayerHandler {
	return GetPlayerHandler{useCase: useCase}
}

type GetPlayerHandler struct {
	useCase UseCase
}

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

	res := connect.NewResponse(&playerv1.GetPlayerResponse{Player: playermessage.Player(player.Player, player.Titles)})
	res.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
	return res, nil
}
