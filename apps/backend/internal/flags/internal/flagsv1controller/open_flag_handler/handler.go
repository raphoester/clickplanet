package open_flag_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/open_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/flagmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
)

type UseCase interface {
	Execute(ctx context.Context, in open_flag_usecase.In) (catalog.Snapshot, error)
}

func New(useCase UseCase, messages flagmessage.Messages) OpenFlagHandler {
	return OpenFlagHandler{useCase: useCase, messages: messages}
}

type OpenFlagHandler struct {
	useCase  UseCase
	messages flagmessage.Messages
}

func (h OpenFlagHandler) OpenFlag(
	ctx context.Context,
	req *connect.Request[flagsv1.OpenFlagRequest],
) (*connect.Response[flagsv1.OpenFlagResponse], error) {
	if req.Msg.GetPin() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, catalog.ErrNoPin)
	}
	flag, err := h.useCase.Execute(ctx, open_flag_usecase.In{
		Name: req.Msg.GetName(), Upload: req.Msg.GetUploadId(), Latitude: req.Msg.GetPin().GetLatitude(),
		Longitude: req.Msg.GetPin().GetLongitude(), EndsAt: req.Msg.GetEndsAt().AsTime(),
	})
	switch {
	case errors.Is(err, catalog.ErrInvalidName), errors.Is(err, catalog.ErrEnded), errors.Is(err, catalog.ErrInvalidPin),
		errors.Is(err, pictures.ErrUnknownUpload), errors.Is(err, pictures.ErrTooLarge),
		errors.Is(err, pictures.ErrNotAPicture), errors.Is(err, pictures.ErrMoves),
		errors.Is(err, pictures.ErrShape), errors.Is(err, pictures.ErrTooSmall):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, pictures.ErrOff):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&flagsv1.OpenFlagResponse{Flag: h.messages.Flag(flag)}), nil
}
