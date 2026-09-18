// Package claim_bonus_handler serves planet.v1.ClickService/ClaimBonus.
package claim_bonus_handler

import (
	"context"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/claim_bonus_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/chargesheld"
)

type UseCase interface {
	Execute(ctx context.Context, in claim_bonus_usecase.In) (claim_bonus_usecase.Out, error)
}

func New(useCase UseCase) ClaimBonusHandler {
	return ClaimBonusHandler{useCase: useCase}
}

type ClaimBonusHandler struct {
	useCase UseCase
}

func (h ClaimBonusHandler) ClaimBonus(
	ctx context.Context,
	req *connect.Request[planetv1.ClaimBonusRequest],
) (*connect.Response[planetv1.ClaimBonusResponse], error) {
	out, err := h.useCase.Execute(ctx, claim_bonus_usecase.In{
		Token:     req.Msg.GetToken(),
		CountryID: req.Msg.GetCountryId(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, claim_bonus_usecase.ErrNoSuchBonus)
	}

	return connect.NewResponse(&planetv1.ClaimBonusResponse{
		Kind:    EncodeKind(out.Kind),
		Amount:  uint32(out.Amount),
		Charges: chargesheld.Encode(out.Held),
	}), nil
}

func EncodeKind(kind bonuses.Kind) planetv1.BonusKind {
	switch kind {
	case bonuses.KindRefill:
		return planetv1.BonusKind_BONUS_KIND_REFILL
	case bonuses.KindSpreadClicks:
		return planetv1.BonusKind_BONUS_KIND_SPREAD_CLICKS
	case bonuses.KindBomb:
		return planetv1.BonusKind_BONUS_KIND_BOMB
	case bonuses.KindEncloseClicks:
		return planetv1.BonusKind_BONUS_KIND_ENCLOSE_CLICKS
	}

	return planetv1.BonusKind_BONUS_KIND_UNSPECIFIED
}
