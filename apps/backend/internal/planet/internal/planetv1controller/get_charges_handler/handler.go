package get_charges_handler

import (
	"context"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/chargesheld"
)

type UseCase interface {
	Execute(ctx context.Context) (bonuses.Held, error)
}

func New(useCase UseCase) GetChargesHandler {
	return GetChargesHandler{useCase: useCase}
}

type GetChargesHandler struct {
	useCase UseCase
}

func (h GetChargesHandler) GetCharges(
	ctx context.Context,
	_ *connect.Request[planetv1.GetChargesRequest],
) (*connect.Response[planetv1.GetChargesResponse], error) {
	held, err := h.useCase.Execute(ctx)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&planetv1.GetChargesResponse{Charges: chargesheld.Encode(held)}), nil
}
