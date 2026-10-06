package grant_charges_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/grant_charges_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/chargesheld"
)

type UseCase interface {
	Execute(ctx context.Context, in grant_charges_usecase.In) (grant_charges_usecase.Out, error)
}

func New(useCase UseCase) GrantChargesHandler {
	return GrantChargesHandler{useCase: useCase}
}

type GrantChargesHandler struct {
	useCase UseCase
}

func (h GrantChargesHandler) GrantCharges(
	ctx context.Context,
	req *connect.Request[planetv1.GrantChargesRequest],
) (*connect.Response[planetv1.GrantChargesResponse], error) {
	out, err := h.useCase.Execute(ctx, grant_charges_usecase.In{
		Account: req.Msg.GetAccountId(),
		Grant: bonuses.Held{
			Refill:       req.Msg.GetRefill(),
			Bomb:         req.Msg.GetBomb(),
			Enclosures:   int(req.Msg.GetEnclosures()),
			SpreadClicks: int(req.Msg.GetSpreadClicks()),
			Defenders:    int(req.Msg.GetDefenders()),
		},
	})

	switch {
	case err == nil:
		return connect.NewResponse(&planetv1.GrantChargesResponse{
			AccountId: string(out.Holder),
			Before:    chargesheld.Encode(out.Before),
			After:     chargesheld.Encode(out.After),
		}), nil
	case errors.Is(err, bonuses.ErrInvalidHolder), errors.Is(err, bonuses.ErrNothingToGrant),
		errors.Is(err, bonuses.ErrNegativeGrant):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return nil, err
	}
}
