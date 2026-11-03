package rename_player_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/rename_player_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in rename_player_usecase.In) (players.Profile, error)
}

func New(useCase UseCase) RenamePlayerHandler {
	return RenamePlayerHandler{useCase: useCase}
}

type RenamePlayerHandler struct {
	useCase UseCase
}

func (h RenamePlayerHandler) RenamePlayer(
	ctx context.Context,
	req *connect.Request[playerv1.RenamePlayerRequest],
) (*connect.Response[playerv1.RenamePlayerResponse], error) {
	account, err := players.AccountIDOf(req.Msg.GetAccountId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	profile, err := h.useCase.Execute(ctx, rename_player_usecase.In{Account: account, Name: req.Msg.GetName()})
	switch {
	case errors.Is(err, players.ErrInvalidName):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, players.ErrNoProfile):
		return nil, connect.NewError(connect.CodeNotFound, players.ErrNoProfile)
	case errors.Is(err, players.ErrNameTaken):
		return nil, connect.NewError(connect.CodeAlreadyExists, players.ErrNameTaken)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	return connect.NewResponse(&playerv1.RenamePlayerResponse{Name: string(profile.TakeSnapshot().Name)}), nil
}
