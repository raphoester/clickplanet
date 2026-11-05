package get_season_totals_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

type Query interface {
	Totals(ctx context.Context, number season.Number) (*planetv1.LedgerTotals, error)
}

func New(query Query) GetSeasonTotalsHandler {
	return GetSeasonTotalsHandler{query: query}
}

type GetSeasonTotalsHandler struct {
	query Query
}

func (h GetSeasonTotalsHandler) GetSeasonTotals(
	ctx context.Context,
	req *connect.Request[gamemasterv1.GetSeasonTotalsRequest],
) (*connect.Response[gamemasterv1.GetSeasonTotalsResponse], error) {
	totals, err := h.query.Totals(ctx, season.Number(req.Msg.GetSeason()))
	if errors.Is(err, season.ErrNotEnded) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&gamemasterv1.GetSeasonTotalsResponse{Totals: totals}), nil
}
