package place_shield_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/place_shield_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/chargesheld"
)

type UseCase interface {
	Execute(ctx context.Context, in place_shield_usecase.In) (bonuses.Held, error)
}

func New(useCase UseCase) PlaceShieldHandler {
	return PlaceShieldHandler{useCase: useCase}
}

type PlaceShieldHandler struct {
	useCase UseCase
}

func (h PlaceShieldHandler) PlaceShield(
	ctx context.Context,
	req *connect.Request[planetv1.PlaceShieldRequest],
) (*connect.Response[planetv1.PlaceShieldResponse], error) {
	held, err := h.useCase.Execute(ctx, place_shield_usecase.In{
		TileID:    req.Msg.GetTileId(),
		CountryID: req.Msg.GetCountryId(),
	})

	switch {
	case err == nil:
		return connect.NewResponse(&planetv1.PlaceShieldResponse{Charges: chargesheld.Encode(held)}), nil
	case errors.Is(err, clicks.ErrUnknownCountry), errors.Is(err, clicks.ErrTileOutOfRange):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, clicks.ErrNotYourTile), errors.Is(err, clicks.ErrTileFull):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, place_shield_usecase.ErrNoShield):
		return nil, connect.NewError(connect.CodeNotFound, err)
	default:
		return nil, err
	}
}
