package delete_season_handler

import (
	"context"

	"connectrpc.com/connect"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/seasonmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

type UseCase interface {
	Execute(ctx context.Context, number season.Number) error
}

func New(useCase UseCase) DeleteSeasonHandler {
	return DeleteSeasonHandler{useCase: useCase}
}

type DeleteSeasonHandler struct {
	useCase UseCase
}

func (h DeleteSeasonHandler) DeleteSeason(
	ctx context.Context,
	req *connect.Request[gamemasterv1.DeleteSeasonRequest],
) (*connect.Response[gamemasterv1.DeleteSeasonResponse], error) {
	if err := h.useCase.Execute(ctx, season.Number(req.Msg.GetNumber())); err != nil {
		return nil, seasonmessage.Refusal(err) //nolint:wrapcheck // the error net answers what is not the operator's.
	}
	return connect.NewResponse(&gamemasterv1.DeleteSeasonResponse{}), nil
}
