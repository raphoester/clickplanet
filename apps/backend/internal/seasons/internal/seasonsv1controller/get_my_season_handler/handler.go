package get_my_season_handler

import (
	"context"

	"connectrpc.com/connect"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
)

type Query interface {
	MySeason(ctx context.Context, account standings.AccountID) (*seasonsv1.GetMySeasonResponse, error)
}

func New(query Query) GetMySeasonHandler {
	return GetMySeasonHandler{query: query}
}

type GetMySeasonHandler struct {
	query Query
}

func (h GetMySeasonHandler) GetMySeason(
	ctx context.Context,
	_ *connect.Request[seasonsv1.GetMySeasonRequest],
) (*connect.Response[seasonsv1.GetMySeasonResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	mine, err := h.query.MySeason(ctx, account)
	if err != nil {
		return nil, err //nolint:wrapcheck // a storage failure is the error net's to answer.
	}
	return connect.NewResponse(mine), nil
}
