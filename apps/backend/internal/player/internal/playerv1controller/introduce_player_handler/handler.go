package introduce_player_handler

import (
	"context"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
)

type UseCase interface {
	Execute(ctx context.Context, account players.AccountID) (players.Player, error)
}

func New(useCase UseCase) IntroducePlayerHandler {
	return IntroducePlayerHandler{useCase: useCase}
}

type IntroducePlayerHandler struct {
	useCase UseCase
}

func (h IntroducePlayerHandler) IntroducePlayer(
	ctx context.Context,
	req *connect.Request[playerv1.IntroducePlayerRequest],
) (*connect.Response[playerv1.IntroducePlayerResponse], error) {
	account, err := players.AccountIDOf(req.Msg.GetAccountId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	player, err := h.useCase.Execute(ctx, account)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}

	return connect.NewResponse(&playerv1.IntroducePlayerResponse{Player: &playerv1.Player{
		AccountId: account.String(),
		Name:      player.Name(),
		Admin:     player.Admin(),
		Color:     playermessage.Color(player.Color()),
		Guest:     player.Guest(),
	}}), nil
}
