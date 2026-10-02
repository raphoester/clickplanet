// Package use_refill_handler serves planet.v1.ClickService/UseRefill.
package use_refill_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/use_refill_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/chargesheld"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/clickbudget"
)

type UseCase interface {
	Execute(ctx context.Context, in use_refill_usecase.In) (use_refill_usecase.Out, error)
}

func New(useCase UseCase) UseRefillHandler {
	return UseRefillHandler{useCase: useCase}
}

type UseRefillHandler struct {
	useCase UseCase
}

func (h UseRefillHandler) UseRefill(
	ctx context.Context,
	req *connect.Request[planetv1.UseRefillRequest],
) (*connect.Response[planetv1.UseRefillResponse], error) {
	out, err := h.useCase.Execute(ctx, use_refill_usecase.In{CountryID: req.Msg.GetCountryId()})

	switch {
	case err == nil:
		return connect.NewResponse(&planetv1.UseRefillResponse{
			Budget:  clickbudget.Encode(out.Budget),
			Charges: chargesheld.Encode(out.Held),
		}), nil
	case errors.Is(err, use_refill_usecase.ErrNoRefill):
		return nil, connect.NewError(connect.CodeNotFound, use_refill_usecase.ErrNoRefill)
	case errors.Is(err, use_refill_usecase.ErrBankFull):
		return nil, connect.NewError(connect.CodeFailedPrecondition, use_refill_usecase.ErrBankFull)
	default:
		return nil, err
	}
}
