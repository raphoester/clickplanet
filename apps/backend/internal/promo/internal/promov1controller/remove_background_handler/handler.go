package remove_background_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds"
)

type UseCase interface {
	Execute(ctx context.Context, id backgrounds.BackgroundID) error
}

func New(useCase UseCase) RemoveBackgroundHandler {
	return RemoveBackgroundHandler{useCase: useCase}
}

type RemoveBackgroundHandler struct {
	useCase UseCase
}

func (h RemoveBackgroundHandler) RemoveBackground(
	ctx context.Context,
	req *connect.Request[promov1.RemoveBackgroundRequest],
) (*connect.Response[promov1.RemoveBackgroundResponse], error) {
	id, err := backgrounds.BackgroundIDOf(req.Msg.GetBackgroundId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	err = h.useCase.Execute(ctx, id)
	switch {
	case errors.Is(err, backgrounds.ErrNoBackground):
		return nil, connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, backgrounds.ErrShown):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, backgrounds.ErrBackgroundsOff):
		return nil, connect.NewError(connect.CodeFailedPrecondition, backgrounds.ErrBackgroundsOff)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&promov1.RemoveBackgroundResponse{}), nil
}
