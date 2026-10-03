package get_titles_handler

import (
	"context"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/get_titles_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, account players.AccountID) (get_titles_usecase.Dashboard, error)
}

func New(useCase UseCase) GetTitlesHandler {
	return GetTitlesHandler{useCase: useCase}
}

type GetTitlesHandler struct {
	useCase UseCase
}

func (h GetTitlesHandler) GetTitles(
	ctx context.Context,
	_ *connect.Request[playerv1.GetTitlesRequest],
) (*connect.Response[playerv1.GetTitlesResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	dashboard, err := h.useCase.Execute(ctx, account)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	return connect.NewResponse(playermessage.Dashboard(dashboard.Showcase, dashboard.Tracks)), nil
}
