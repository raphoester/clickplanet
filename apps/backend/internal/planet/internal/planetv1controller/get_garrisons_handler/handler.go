package get_garrisons_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/garrisons"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/garrisonmessage"
)

type UseCase interface {
	Execute(ctx context.Context) []garrisons.Garrison
}

func New(useCase UseCase) GetGarrisonsHandler {
	return GetGarrisonsHandler{useCase: useCase}
}

type GetGarrisonsHandler struct {
	useCase UseCase
}

func (h GetGarrisonsHandler) GetGarrisons(
	ctx context.Context,
	_ *connect.Request[planetv1.GetGarrisonsRequest],
) (*connect.Response[planetv1.GetGarrisonsResponse], error) {
	standing := h.useCase.Execute(ctx)
	out := make([]*planetv1.Garrison, len(standing))
	for i, garrison := range standing {
		out[i] = garrisonmessage.Encode(garrison)
	}

	return connect.NewResponse(&planetv1.GetGarrisonsResponse{Garrisons: out}), nil
}
