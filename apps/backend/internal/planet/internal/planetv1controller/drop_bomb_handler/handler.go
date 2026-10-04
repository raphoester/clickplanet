package drop_bomb_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/drop_bomb_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/frozenmap"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
)

type UseCase interface {
	Execute(ctx context.Context, in drop_bomb_usecase.In) (clicks.Blast, error)
}

func New(useCase UseCase) DropBombHandler {
	return DropBombHandler{useCase: useCase}
}

type DropBombHandler struct {
	useCase UseCase
}

func (h DropBombHandler) DropBomb(
	ctx context.Context,
	req *connect.Request[planetv1.DropBombRequest],
) (*connect.Response[planetv1.DropBombResponse], error) {
	target := req.Msg.GetTarget()
	_, err := h.useCase.Execute(ctx, drop_bomb_usecase.In{
		Target:    clicks.Vec3{X: target.GetX(), Y: target.GetY(), Z: target.GetZ()},
		CountryID: req.Msg.GetCountryId(),
	})

	switch {
	case err == nil:
		return connect.NewResponse(&planetv1.DropBombResponse{}), nil
	case errors.Is(err, clicks.ErrUnknownCountry), errors.Is(err, clicks.ErrTileOutOfRange):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, drop_bomb_usecase.ErrNoBomb):
		return nil, connect.NewError(connect.CodeNotFound, drop_bomb_usecase.ErrNoBomb)
	case errors.Is(err, tempo.ErrFrozen):
		return nil, frozenmap.Refusal(err)
	default:
		return nil, err
	}
}
