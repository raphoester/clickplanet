package get_my_season_handler

import (
	"context"

	"connectrpc.com/connect"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
)

type UseCase interface {
	Execute(ctx context.Context, account standings.AccountID) (standings.Place, error)
}

func New(useCase UseCase) GetMySeasonHandler {
	return GetMySeasonHandler{useCase: useCase}
}

type GetMySeasonHandler struct {
	useCase UseCase
}

func (h GetMySeasonHandler) GetMySeason(
	ctx context.Context,
	_ *connect.Request[seasonsv1.GetMySeasonRequest],
) (*connect.Response[seasonsv1.GetMySeasonResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	place, err := h.useCase.Execute(ctx, account)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	return connect.NewResponse(&seasonsv1.GetMySeasonResponse{
		CountryId:   string(place.Line.Country),
		Tiles:       place.Line.Tiles,
		GlobalRank:  place.GlobalRank,
		CountryRank: place.CountryRank,
	}), nil
}
