// Package map_density_handler serves planet.v1.ClickService/MapDensity.
package map_density_handler

import (
	"context"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

type UseCase interface {
	Execute(ctx context.Context) uint32
}

func New(useCase UseCase) MapDensityHandler {
	return MapDensityHandler{useCase: useCase}
}

type MapDensityHandler struct {
	useCase UseCase
}

func (h MapDensityHandler) MapDensity(
	ctx context.Context,
	_ *connect.Request[planetv1.MapDensityRequest],
) (*connect.Response[planetv1.MapDensityResponse], error) {
	return connect.NewResponse(&planetv1.MapDensityResponse{
		Density: h.useCase.Execute(ctx),
	}), nil
}
