package close_flag_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/close_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/flagmessage"
)

type UseCase interface {
	Execute(ctx context.Context, in close_flag_usecase.In) (catalog.Snapshot, error)
}

func New(useCase UseCase, messages flagmessage.Messages) CloseFlagHandler {
	return CloseFlagHandler{useCase: useCase, messages: messages}
}

type CloseFlagHandler struct {
	useCase  UseCase
	messages flagmessage.Messages
}

func (h CloseFlagHandler) CloseFlag(
	ctx context.Context,
	req *connect.Request[flagsv1.CloseFlagRequest],
) (*connect.Response[flagsv1.CloseFlagResponse], error) {
	flag, err := h.useCase.Execute(ctx, close_flag_usecase.In{Flag: req.Msg.GetCode(), Note: req.Msg.GetNote()})
	switch {
	case errors.Is(err, catalog.ErrInvalidFlagID), errors.Is(err, catalog.ErrInvalidNote):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, catalog.ErrUnknownFlag):
		return nil, connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, catalog.ErrClosed):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&flagsv1.CloseFlagResponse{Flag: h.messages.Flag(flag)}), nil
}
