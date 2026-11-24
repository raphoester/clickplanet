package rename_flag_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/rename_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/flagmessage"
)

type UseCase interface {
	Execute(ctx context.Context, in rename_flag_usecase.In) (catalog.Snapshot, error)
}

func New(useCase UseCase, messages flagmessage.Messages) RenameFlagHandler {
	return RenameFlagHandler{useCase: useCase, messages: messages}
}

type RenameFlagHandler struct {
	useCase  UseCase
	messages flagmessage.Messages
}

func (h RenameFlagHandler) RenameFlag(
	ctx context.Context,
	req *connect.Request[flagsv1.RenameFlagRequest],
) (*connect.Response[flagsv1.RenameFlagResponse], error) {
	flag, err := h.useCase.Execute(ctx, rename_flag_usecase.In{Flag: req.Msg.GetCode(), Name: req.Msg.GetName()})
	switch {
	case errors.Is(err, catalog.ErrInvalidFlagID), errors.Is(err, catalog.ErrInvalidName),
		errors.Is(err, catalog.ErrNameTaken):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, catalog.ErrUnknownFlag):
		return nil, connect.NewError(connect.CodeNotFound, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&flagsv1.RenameFlagResponse{Flag: h.messages.Flag(flag)}), nil
}
