package grant_charge_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/grant_charges_usecase"
)

var ErrUnknownKind = errors.New("not a kind of charge")

type UseCase interface {
	Execute(ctx context.Context, in grant_charges_usecase.In) (grant_charges_usecase.Out, error)
}

func New(useCase UseCase) GrantChargeHandler {
	return GrantChargeHandler{useCase: useCase}
}

type GrantChargeHandler struct {
	useCase UseCase
}

func (h GrantChargeHandler) GrantCharge(
	ctx context.Context,
	req *connect.Request[planetv1.GrantChargeRequest],
) (*connect.Response[planetv1.GrantChargeResponse], error) {
	kind, err := kindOf(req.Msg.GetKind())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	out, err := h.useCase.Execute(ctx, grant_charges_usecase.In{
		Account: req.Msg.GetAccountId(),
		Grant:   bonuses.GrantOf(kind, int(req.Msg.GetAmount())),
	})
	switch {
	case errors.Is(err, bonuses.ErrInvalidHolder), errors.Is(err, bonuses.ErrNothingToGrant):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	kept := out.After.Count(kind) - out.Before.Count(kind)

	return connect.NewResponse(&planetv1.GrantChargeResponse{Kept: uint32(max(kept, 0))}), nil //nolint:gosec // a stack's size.
}

func kindOf(kind planetv1.BonusKind) (bonuses.Kind, error) {
	switch kind {
	case planetv1.BonusKind_BONUS_KIND_REFILL:
		return bonuses.KindRefill, nil
	case planetv1.BonusKind_BONUS_KIND_SPREAD_CLICKS:
		return bonuses.KindSpreadClicks, nil
	case planetv1.BonusKind_BONUS_KIND_BOMB:
		return bonuses.KindBomb, nil
	case planetv1.BonusKind_BONUS_KIND_ENCLOSE_CLICKS:
		return bonuses.KindEncloseClicks, nil
	case planetv1.BonusKind_BONUS_KIND_SHIELDS:
		return bonuses.KindShields, nil
	case planetv1.BonusKind_BONUS_KIND_UNSPECIFIED:
	}

	return "", fmt.Errorf("%w: %v", ErrUnknownKind, kind)
}
