package get_missions_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/missions/usecases/get_missions_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
)

type UseCase interface {
	Execute(ctx context.Context, in get_missions_usecase.In) (get_missions_usecase.Out, error)
}

func New(useCase UseCase) GetMissionsHandler {
	return GetMissionsHandler{useCase: useCase}
}

type GetMissionsHandler struct {
	useCase UseCase
}

func (h GetMissionsHandler) GetMissions(
	ctx context.Context,
	req *connect.Request[playerv1.GetMissionsRequest],
) (*connect.Response[playerv1.GetMissionsResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	out, err := h.useCase.Execute(ctx, get_missions_usecase.In{Account: account, Country: req.Msg.GetCountryId()})
	switch {
	case errors.Is(err, get_missions_usecase.ErrUnknownCountry):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	sheet := out.Sheet.Snapshot()
	answer := &playerv1.GetMissionsResponse{
		CountryId: out.Flag, Enlisted: out.Sheet.Enlisted(), EndsAtUnixMs: out.Day.EndsAt().UnixMilli(),
	}
	for _, mission := range out.Brief.Snapshot() {
		answer.Missions = append(answer.Missions, playermessage.Mission(mission, sheet))
	}
	return connect.NewResponse(answer), nil
}
