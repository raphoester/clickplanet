package keep_background_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds/usecases/keep_background_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in keep_background_usecase.In) error
}

func New(useCase UseCase) KeepBackgroundHandler {
	return KeepBackgroundHandler{useCase: useCase}
}

type KeepBackgroundHandler struct {
	useCase UseCase
}

func (h KeepBackgroundHandler) KeepBackground(
	ctx context.Context,
	req *connect.Request[promov1.KeepBackgroundRequest],
) (*connect.Response[promov1.KeepBackgroundResponse], error) {
	id, err := backgrounds.BackgroundIDOf(req.Msg.GetBackgroundId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	err = h.useCase.Execute(ctx, keep_background_usecase.In{ID: id, Name: req.Msg.GetName()})
	switch {
	case errors.Is(err, backgrounds.ErrInvalidBackground):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, backgrounds.ErrNotUploaded):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, backgrounds.ErrBackgroundsOff):
		return nil, connect.NewError(connect.CodeFailedPrecondition, backgrounds.ErrBackgroundsOff)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&promov1.KeepBackgroundResponse{}), nil
}
