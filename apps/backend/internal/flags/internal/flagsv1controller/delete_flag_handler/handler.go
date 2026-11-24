package delete_flag_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/delete_flag_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in delete_flag_usecase.In) error
}

func New(useCase UseCase) DeleteFlagHandler {
	return DeleteFlagHandler{useCase: useCase}
}

type DeleteFlagHandler struct {
	useCase UseCase
}

func (h DeleteFlagHandler) DeleteFlag(
	ctx context.Context,
	req *connect.Request[flagsv1.DeleteFlagRequest],
) (*connect.Response[flagsv1.DeleteFlagResponse], error) {
	err := h.useCase.Execute(ctx, delete_flag_usecase.In{Flag: req.Msg.GetCode()})
	switch {
	case errors.Is(err, catalog.ErrInvalidFlagID):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, catalog.ErrUnknownFlag):
		return nil, connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, catalog.ErrOpen), errors.Is(err, catalog.ErrPlayed), errors.Is(err, catalog.ErrJustClosed):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&flagsv1.DeleteFlagResponse{}), nil
}
