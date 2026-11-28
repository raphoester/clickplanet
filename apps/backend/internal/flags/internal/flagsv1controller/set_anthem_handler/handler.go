package set_anthem_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/set_anthem_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type UseCase interface {
	Execute(ctx context.Context, in set_anthem_usecase.In) error
}

func New(useCase UseCase) SetAnthemHandler {
	return SetAnthemHandler{useCase: useCase}
}

type SetAnthemHandler struct {
	useCase UseCase
}

// A guest holds no flag: its account is no holder, and the rule answers PermissionDenied.
func (h SetAnthemHandler) SetAnthem(
	ctx context.Context,
	req *connect.Request[flagsv1.SetAnthemRequest],
) (*connect.Response[flagsv1.SetAnthemResponse], error) {
	err := h.useCase.Execute(ctx, set_anthem_usecase.In{
		Account: cpctx.GetAccount(ctx), Flag: req.Msg.GetFlag(), Anthem: req.Msg.GetAnthem().GetLibraryCode(),
	})
	switch {
	case errors.Is(err, catalog.ErrNotHolder), errors.Is(err, set_anthem_usecase.ErrNoAccount):
		return nil, connect.NewError(connect.CodePermissionDenied, catalog.ErrNotHolder)
	case errors.Is(err, catalog.ErrRemoved):
		return nil, connect.NewError(connect.CodeFailedPrecondition, catalog.ErrRemoved)
	case errors.Is(err, catalog.ErrInvalidAnthem), errors.Is(err, catalog.ErrInvalidFlagID):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, catalog.ErrUnknownFlag):
		return nil, connect.NewError(connect.CodeNotFound, catalog.ErrUnknownFlag)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&flagsv1.SetAnthemResponse{}), nil
}
