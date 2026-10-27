package get_my_season_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_my_season_handler/my_season_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings"
)

type Query interface {
	MySeason(ctx context.Context, account standings.AccountID, country string) (*scoringv1.GetMySeasonResponse, error)
}

func New(query Query) GetMySeasonHandler {
	return GetMySeasonHandler{query: query}
}

type GetMySeasonHandler struct {
	query Query
}

func (h GetMySeasonHandler) GetMySeason(
	ctx context.Context,
	req *connect.Request[scoringv1.GetMySeasonRequest],
) (*connect.Response[scoringv1.GetMySeasonResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	mine, err := h.query.MySeason(ctx, account, req.Msg.GetCountryId())
	if errors.Is(err, my_season_query.ErrUnknownCountry) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // a storage failure is the error net's to answer.
	}
	return connect.NewResponse(mine), nil
}
