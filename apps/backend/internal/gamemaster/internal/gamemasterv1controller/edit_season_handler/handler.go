package edit_season_handler

import (
	"context"

	"connectrpc.com/connect"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/seasonmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

type UseCase interface {
	Execute(ctx context.Context, number season.Number, draft season.Draft) error
}

func New(useCase UseCase) EditSeasonHandler {
	return EditSeasonHandler{useCase: useCase}
}

type EditSeasonHandler struct {
	useCase UseCase
}

func (h EditSeasonHandler) EditSeason(
	ctx context.Context,
	req *connect.Request[gamemasterv1.EditSeasonRequest],
) (*connect.Response[gamemasterv1.EditSeasonResponse], error) {
	draft := seasonmessage.DraftOf(req.Msg.GetStartsAt(), req.Msg.GetEndsAt())
	if err := h.useCase.Execute(ctx, season.Number(req.Msg.GetNumber()), draft); err != nil {
		return nil, seasonmessage.Refusal(err) //nolint:wrapcheck // the error net answers what is not the operator's.
	}
	return connect.NewResponse(&gamemasterv1.EditSeasonResponse{}), nil
}
