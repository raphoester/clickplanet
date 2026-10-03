package get_season_handler

import (
	"context"

	"connectrpc.com/connect"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
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
	_ *connect.Request[seasonsv1.GetSeasonRequest],
) (*connect.Response[seasonsv1.GetSeasonResponse], error) {
	season, ok := h.useCase.Execute(ctx)
	if !ok {
		return connect.NewResponse(&seasonsv1.GetSeasonResponse{}), nil
	}

	return connect.NewResponse(&seasonsv1.GetSeasonResponse{Season: &seasonsv1.Season{
		Number:               uint32(season.Number),
		FinaleStartsAtUnixMs: season.FinaleStartsAt.UnixMilli(),
		EndsAtUnixMs:         season.EndsAt.UnixMilli(),
	}}), nil
}
