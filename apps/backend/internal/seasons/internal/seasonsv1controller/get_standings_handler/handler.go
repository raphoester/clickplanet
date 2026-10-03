package get_standings_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
)

type UseCase interface {
	Execute(ctx context.Context, country standings.Country) ([]standings.Standing, error)
}

func New(useCase UseCase) GetStandingsHandler {
	return GetStandingsHandler{useCase: useCase}
}

type GetStandingsHandler struct {
	useCase UseCase
}

func (h GetStandingsHandler) GetStandings(
	ctx context.Context,
	req *connect.Request[seasonsv1.GetStandingsRequest],
) (*connect.Response[seasonsv1.GetStandingsResponse], error) {
	top, err := h.useCase.Execute(ctx, standings.Country(req.Msg.GetCountryId()))
	if errors.Is(err, standings.ErrUnknownCountry) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	lines := make([]*seasonsv1.Standing, len(top))
	for i, standing := range top {
		lines[i] = &seasonsv1.Standing{
			Rank:      standing.Rank,
			Name:      standing.Player.Name,
			Color:     playerv1.NameColor(standing.Player.Color),
			CountryId: string(standing.Line.Country),
			Tiles:     standing.Line.Tiles,
		}
	}
	return connect.NewResponse(&seasonsv1.GetStandingsResponse{Standings: lines}), nil
}
