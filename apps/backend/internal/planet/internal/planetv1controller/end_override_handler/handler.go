package end_override_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

type UseCase interface {
	Execute(ctx context.Context) error
}

func New(useCase UseCase) EndOverrideHandler {
	return EndOverrideHandler{useCase: useCase}
}

type EndOverrideHandler struct {
	useCase UseCase
}

func (h EndOverrideHandler) EndOverride(
	ctx context.Context,
	_ *connect.Request[planetv1.EndOverrideRequest],
) (*connect.Response[planetv1.EndOverrideResponse], error) {
	if err := h.useCase.Execute(ctx); err != nil {
		return nil, err //nolint:wrapcheck // the use case says what failed.
	}

	return connect.NewResponse(&planetv1.EndOverrideResponse{}), nil
}
