package claim_bonus_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/claim_bonus_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/planetwords"
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
	req *connect.Request[bonusesv1.ClaimBonusRequest],
) (*connect.Response[bonusesv1.ClaimBonusResponse], error) {
	out, err := h.useCase.Execute(ctx, claim_bonus_usecase.In{
		Caller:    caller.Of(ctx),
		Token:     req.Msg.GetToken(),
		CountryID: req.Msg.GetCountryId(),
	})
	if errors.Is(err, claim_bonus_usecase.ErrNoSuchBonus) {
		return nil, connect.NewError(connect.CodeNotFound, claim_bonus_usecase.ErrNoSuchBonus)
	}
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&bonusesv1.ClaimBonusResponse{
		Kind:    planetwords.Kind(out.Kind),
		Amount:  uint32(out.Amount),
		Charges: planetwords.Hand(out.Hand),
	}), nil
}
