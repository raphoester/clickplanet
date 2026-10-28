package add_season_handler

import (
	"context"

	"connectrpc.com/connect"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/seasonmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

type UseCase interface {
	Execute(ctx context.Context, draft season.Draft) (season.Number, error)
}

func New(useCase UseCase) AddSeasonHandler {
	return AddSeasonHandler{useCase: useCase}
}

type AddSeasonHandler struct {
	useCase UseCase
}

func (h AddSeasonHandler) AddSeason(
	ctx context.Context,
	req *connect.Request[gamemasterv1.AddSeasonRequest],
) (*connect.Response[gamemasterv1.AddSeasonResponse], error) {
	number, err := h.useCase.Execute(ctx, seasonmessage.DraftOf(req.Msg.GetBreakBefore(), req.Msg.GetLength()))
	if err != nil {
		return nil, seasonmessage.Refusal(err) //nolint:wrapcheck // the error net answers what is not the operator's.
	}
	return connect.NewResponse(&gamemasterv1.AddSeasonResponse{Number: uint32(number)}), nil
}
