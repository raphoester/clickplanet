package get_season_handler

import (
	"context"

	"connectrpc.com/connect"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar"
)

type UseCase interface {
	Execute(ctx context.Context) (calendar.Season, bool)
}

func New(useCase UseCase) GetSeasonHandler {
	return GetSeasonHandler{useCase: useCase}
}

type GetSeasonHandler struct {
	useCase UseCase
}

func (h GetSeasonHandler) GetSeason(
	ctx context.Context,
	_ *connect.Request[scoringv1.GetSeasonRequest],
) (*connect.Response[scoringv1.GetSeasonResponse], error) {
	season, ok := h.useCase.Execute(ctx)
	if !ok {
		return connect.NewResponse(&scoringv1.GetSeasonResponse{}), nil
	}

	return connect.NewResponse(&scoringv1.GetSeasonResponse{Season: &scoringv1.Season{
		Number:               uint32(season.Number),
		FinaleStartsAtUnixMs: season.LastDayEndsAt.UnixMilli(),
		EndsAtUnixMs:         season.EndsAt.UnixMilli(),
		LastDayEndsAtUnixMs:  season.LastDayEndsAt.UnixMilli(),
	}}), nil
}
