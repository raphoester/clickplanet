// Package get_charges_handler serves planet.v1.ClickService/GetCharges.
package get_charges_handler

import (
	"context"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/chargesheld"
)

type UseCase interface {
	Execute(ctx context.Context) bonuses.Held
}

func New(useCase UseCase) GetChargesHandler {
	return GetChargesHandler{useCase: useCase}
}

type GetChargesHandler struct {
	useCase UseCase
}

// GetCharges is how a client that has just loaded, or changed account, learns what it holds.
func (h GetChargesHandler) GetCharges(
	ctx context.Context,
	_ *connect.Request[planetv1.GetChargesRequest],
) (*connect.Response[planetv1.GetChargesResponse], error) {
	return connect.NewResponse(&planetv1.GetChargesResponse{Charges: chargesheld.Encode(h.useCase.Execute(ctx))}), nil
}
