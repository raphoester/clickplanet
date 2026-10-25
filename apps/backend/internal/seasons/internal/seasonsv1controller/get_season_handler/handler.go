package get_season_handler

import (
	"context"

	"connectrpc.com/connect"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
)

type UseCase interface {
	Execute(ctx context.Context) (current, next *calendar.Season)
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
	current, next := h.useCase.Execute(ctx)

	return connect.NewResponse(&seasonsv1.GetSeasonResponse{Season: encode(current), Next: encode(next)}), nil
}

func encode(season *calendar.Season) *seasonsv1.Season {
	if season == nil {
		return nil
	}

	encoded := &seasonsv1.Season{
		Number:               uint32(season.Number),
		FinaleStartsAtUnixMs: season.FinaleStartsAt.UnixMilli(),
		EndsAtUnixMs:         season.EndsAt.UnixMilli(),
	}
	if !season.StartsAt.IsZero() {
		encoded.StartsAtUnixMs = season.StartsAt.UnixMilli()
	}
	return encoded
}
