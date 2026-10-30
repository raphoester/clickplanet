package get_ad_refill_handler

import (
	"context"
	"time"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

type UseCase interface {
	Execute(ctx context.Context) (time.Time, error)
}

func New(useCase UseCase) GetAdRefillHandler {
	return GetAdRefillHandler{useCase: useCase}
}

type GetAdRefillHandler struct {
	useCase UseCase
}

func (h GetAdRefillHandler) GetAdRefill(
	ctx context.Context,
	_ *connect.Request[planetv1.GetAdRefillRequest],
) (*connect.Response[planetv1.GetAdRefillResponse], error) {
	at, err := h.useCase.Execute(ctx)
	if err != nil {
		return nil, err
	}

	answer := &planetv1.GetAdRefillResponse{}
	if !at.IsZero() {
		answer.AvailableAtUnixMs = at.UnixMilli()
	}
	return connect.NewResponse(answer), nil
}
