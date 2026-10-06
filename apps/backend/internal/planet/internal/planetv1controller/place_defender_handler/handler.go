package place_defender_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/garrisons/usecases/place_defender_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/chargesheld"
)

type UseCase interface {
	Execute(ctx context.Context, in place_defender_usecase.In) (bonuses.Held, error)
}

func New(useCase UseCase) PlaceDefenderHandler {
	return PlaceDefenderHandler{useCase: useCase}
}

type PlaceDefenderHandler struct {
	useCase UseCase
}

func (h PlaceDefenderHandler) PlaceDefender(
	ctx context.Context,
	req *connect.Request[planetv1.PlaceDefenderRequest],
) (*connect.Response[planetv1.PlaceDefenderResponse], error) {
	held, err := h.useCase.Execute(ctx, place_defender_usecase.In{
		TileID:    req.Msg.GetTileId(),
		CountryID: req.Msg.GetCountryId(),
	})

	switch {
	case err == nil:
		return connect.NewResponse(&planetv1.PlaceDefenderResponse{Charges: chargesheld.Encode(held)}), nil
	case errors.Is(err, clicks.ErrUnknownCountry), errors.Is(err, clicks.ErrTileOutOfRange):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, place_defender_usecase.ErrNotYours), errors.Is(err, place_defender_usecase.ErrFull):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, place_defender_usecase.ErrNoDefender):
		return nil, connect.NewError(connect.CodeNotFound, err)
	default:
		return nil, err
	}
}
