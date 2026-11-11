package reset_map_handler

import (
	"context"
	"time"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/resets"
)

type UseCase interface {
	Execute(ctx context.Context, reset resets.Reset) (bool, error)
}

func New(useCase UseCase) ResetMapHandler {
	return ResetMapHandler{useCase: useCase}
}

type ResetMapHandler struct {
	useCase UseCase
}

func (h ResetMapHandler) ResetMap(
	ctx context.Context,
	req *connect.Request[planetv1.ResetMapRequest],
) (*connect.Response[planetv1.ResetMapResponse], error) {
	var at time.Time
	if req.Msg.GetDecidedAt() != nil {
		at = req.Msg.GetDecidedAt().AsTime()
	}
	reset, err := resets.New(req.Msg.GetResetId(), at)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	// A caller that gives up never stops the wipe halfway.
	if _, err := h.useCase.Execute(context.WithoutCancel(ctx), reset); err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&planetv1.ResetMapResponse{}), nil
}
