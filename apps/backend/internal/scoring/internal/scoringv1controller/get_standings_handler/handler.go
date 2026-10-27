package get_standings_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_standings_handler/standings_query"
)

type Query interface {
	Standings(ctx context.Context, country string) (*scoringv1.GetStandingsResponse, error)
}

func New(query Query) GetStandingsHandler {
	return GetStandingsHandler{query: query}
}

type GetStandingsHandler struct {
	query Query
}

func (h GetStandingsHandler) GetStandings(
	ctx context.Context,
	req *connect.Request[scoringv1.GetStandingsRequest],
) (*connect.Response[scoringv1.GetStandingsResponse], error) {
	standings, err := h.query.Standings(ctx, req.Msg.GetCountryId())
	if errors.Is(err, standings_query.ErrUnknownCountry) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // a storage failure is the error net's to answer.
	}
	return connect.NewResponse(standings), nil
}
