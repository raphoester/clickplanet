package internal_grant_charges_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/usecases/pay_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/chargesheld"
)

var ErrNoAmount = errors.New("a grant of no charge")

type UseCase interface {
	Execute(ctx context.Context, in pay_usecase.In) (pay_usecase.Out, error)
}

// New takes the payment that knows a grant's id ten minutes, in memory, and the one that keeps it for good.
func New(brief, lasting UseCase) GrantChargesHandler {
	return GrantChargesHandler{brief: brief, lasting: lasting}
}

// GrantChargesHandler grants at once what another module grants an account, once per grant id, and answers what the
// stack kept and the hand after.
type GrantChargesHandler struct {
	brief   UseCase
	lasting UseCase
}

func (h GrantChargesHandler) GrantCharges(
	ctx context.Context, req *connect.Request[planetv1.InternalGrantChargesRequest],
) (*connect.Response[planetv1.InternalGrantChargesResponse], error) {
	grant, err := charges.GrantIDOf(req.Msg.GetGrantId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	holder, err := charges.ParseHolder(req.Msg.GetAccountId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	kind, err := charges.KindOf(req.Msg.GetKind())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if req.Msg.GetAmount() == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrNoAmount)
	}

	pay := h.brief
	if req.Msg.GetLasting() {
		pay = h.lasting
	}
	out, err := pay.Execute(ctx, pay_usecase.In{Grant: grant, Holder: holder, Kind: kind, Amount: int(req.Msg.GetAmount())})
	if err != nil {
		return nil, err //nolint:wrapcheck // the use case named it; the error net keeps it off the wire.
	}

	return connect.NewResponse(&planetv1.InternalGrantChargesResponse{
		Kept:    uint32(out.Kept), //nolint:gosec // what a stack kept, never negative.
		Charges: chargesheld.Encode(out.Held),
	}), nil
}
