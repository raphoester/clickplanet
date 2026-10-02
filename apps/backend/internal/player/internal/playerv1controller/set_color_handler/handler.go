package set_color_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/set_color_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
)

type UseCase interface {
	Execute(ctx context.Context, in set_color_usecase.In) error
}

func New(useCase UseCase) SetColorHandler {
	return SetColorHandler{useCase: useCase}
}

type SetColorHandler struct {
	useCase UseCase
}

var ErrNoUsername = errors.New("only a player with a username may choose a color")

func (h SetColorHandler) SetColor(
	ctx context.Context,
	req *connect.Request[playerv1.SetColorRequest],
) (*connect.Response[playerv1.SetColorResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	color, err := playermessage.ColorOf(req.Msg.GetColor())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	err = h.useCase.Execute(ctx, set_color_usecase.In{Account: account, Color: color})
	switch {
	case errors.Is(err, players.ErrNoProfile):
		return nil, connect.NewError(connect.CodeFailedPrecondition, ErrNoUsername)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	return connect.NewResponse(&playerv1.SetColorResponse{Color: playermessage.Color(color)}), nil
}
