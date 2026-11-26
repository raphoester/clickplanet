package add_time_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/add_time_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/flagmessage"
)

type UseCase interface {
	Execute(ctx context.Context, in add_time_usecase.In) (catalog.Snapshot, error)
}

func New(useCase UseCase, messages flagmessage.Messages) AddTimeHandler {
	return AddTimeHandler{useCase: useCase, messages: messages}
}

type AddTimeHandler struct {
	useCase  UseCase
	messages flagmessage.Messages
}

func (h AddTimeHandler) AddTime(
	ctx context.Context,
	req *connect.Request[flagsv1.AddTimeRequest],
) (*connect.Response[flagsv1.AddTimeResponse], error) {
	flag, err := h.useCase.Execute(ctx, add_time_usecase.In{Flag: req.Msg.GetCode(), Days: req.Msg.GetDays()})
	switch {
	case errors.Is(err, catalog.ErrInvalidFlagID), errors.Is(err, catalog.ErrInvalidDays):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, catalog.ErrUnknownFlag):
		return nil, connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, catalog.ErrNameTaken):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&flagsv1.AddTimeResponse{Flag: h.messages.Flag(flag)}), nil
}
