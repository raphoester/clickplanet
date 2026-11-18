package internal_fill_charges_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/usecases/fill_usecase"
)

var ErrNoKind = errors.New("a fill of no kind")

type UseCase interface {
	Execute(ctx context.Context, in fill_usecase.In) error
}

func New(useCase UseCase) FillChargesHandler {
	return FillChargesHandler{useCase: useCase}
}

// FillChargesHandler fills at once the stacks another module names, once per grant id.
type FillChargesHandler struct {
	useCase UseCase
}

func (h FillChargesHandler) FillCharges(
	ctx context.Context, req *connect.Request[planetv1.FillChargesRequest],
) (*connect.Response[planetv1.FillChargesResponse], error) {
	grant, err := charges.GrantIDOf(req.Msg.GetGrantId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	holder, err := charges.ParseHolder(req.Msg.GetAccountId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if len(req.Msg.GetKinds()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrNoKind)
	}
	kinds := make([]charges.Kind, 0, len(req.Msg.GetKinds()))
	for _, wire := range req.Msg.GetKinds() {
		kind, err := charges.KindOf(wire)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		kinds = append(kinds, kind)
	}

	if err := h.useCase.Execute(ctx, fill_usecase.In{Grant: grant, Holder: holder, Kinds: kinds}); err != nil {
		return nil, err //nolint:wrapcheck // the use case named it; the error net keeps it off the wire.
	}
	return connect.NewResponse(&planetv1.FillChargesResponse{}), nil
}
