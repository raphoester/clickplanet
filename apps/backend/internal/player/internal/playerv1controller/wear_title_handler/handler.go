package wear_title_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
)

type UseCase interface {
	Execute(ctx context.Context, account players.AccountID, title titles.ID) (titles.Standing, error)
}

func New(useCase UseCase) WearTitleHandler {
	return WearTitleHandler{useCase: useCase}
}

type WearTitleHandler struct {
	useCase UseCase
}

func (h WearTitleHandler) WearTitle(
	ctx context.Context,
	req *connect.Request[playerv1.WearTitleRequest],
) (*connect.Response[playerv1.WearTitleResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	worn, err := h.useCase.Execute(ctx, account, titles.ID(req.Msg.GetTitleId()))
	switch {
	case errors.Is(err, titles.ErrNotWearable):
		return nil, connect.NewError(connect.CodeInvalidArgument, titles.ErrNotWearable)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	return connect.NewResponse(&playerv1.WearTitleResponse{Worn: playermessage.Title(worn)}), nil
}
