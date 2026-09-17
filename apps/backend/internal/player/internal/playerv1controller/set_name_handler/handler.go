// Package set_name_handler serves player.v1.PlayerService/SetName.
package set_name_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/set_name_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
)

type UseCase interface {
	Execute(ctx context.Context, in set_name_usecase.In) (players.Profile, error)
}

func New(useCase UseCase) SetNameHandler {
	return SetNameHandler{useCase: useCase}
}

type SetNameHandler struct {
	useCase UseCase
}

func (h SetNameHandler) SetName(
	ctx context.Context,
	req *connect.Request[playerv1.SetNameRequest],
) (*connect.Response[playerv1.SetNameResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	profile, err := h.useCase.Execute(ctx, set_name_usecase.In{Account: account, Name: req.Msg.GetName()})
	switch {
	case errors.Is(err, players.ErrInvalidName):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, players.ErrNameTaken):
		return nil, connect.NewError(connect.CodeAlreadyExists, players.ErrNameTaken)
	case errors.Is(err, players.ErrNotLinked):
		return nil, connect.NewError(connect.CodePermissionDenied, players.ErrNotLinked)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	return connect.NewResponse(&playerv1.SetNameResponse{Profile: playermessage.Profile(profile)}), nil
}
