package fill_charges_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills/usecases/start_fill_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in start_fill_usecase.In) error
}

// New answers for one source: the internal service's handler keeps internal fills, the admin service's an operator's.
func New(useCase UseCase, source fills.Source) FillChargesHandler {
	return FillChargesHandler{useCase: useCase, source: source}
}

type FillChargesHandler struct {
	useCase UseCase
	source  fills.Source
}

func (h FillChargesHandler) FillCharges(
	ctx context.Context,
	req *connect.Request[planetv1.FillChargesRequest],
) (*connect.Response[planetv1.FillChargesResponse], error) {
	err := h.useCase.Execute(ctx, start_fill_usecase.In{
		Request: bonuses.FillID(req.Msg.GetRequestId()),
		Source:  h.source,
	})

	switch {
	case err == nil:
		return connect.NewResponse(&planetv1.FillChargesResponse{}), nil
	case errors.Is(err, fills.ErrNoRequest), errors.Is(err, fills.ErrLongRequest):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return nil, err
	}
}
