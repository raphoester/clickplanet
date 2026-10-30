package claim_ad_refill_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/claim_ad_refill_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/clickbudget"
)

type UseCase interface {
	Execute(ctx context.Context, in claim_ad_refill_usecase.In) (clicks.Budget, error)
}

func New(useCase UseCase) ClaimAdRefillHandler {
	return ClaimAdRefillHandler{useCase: useCase}
}

type ClaimAdRefillHandler struct {
	useCase UseCase
}

func (h ClaimAdRefillHandler) ClaimAdRefill(
	ctx context.Context,
	req *connect.Request[planetv1.ClaimAdRefillRequest],
) (*connect.Response[planetv1.ClaimAdRefillResponse], error) {
	budget, err := h.useCase.Execute(ctx, claim_ad_refill_usecase.In{CountryID: req.Msg.GetCountryId()})

	var used bonuses.AdRefillRefusal
	switch {
	case err == nil:
		return connect.NewResponse(&planetv1.ClaimAdRefillResponse{Budget: clickbudget.Encode(budget)}), nil
	case errors.Is(err, claim_ad_refill_usecase.ErrAdRefillOff):
		return nil, connect.NewError(connect.CodeFailedPrecondition, claim_ad_refill_usecase.ErrAdRefillOff)
	case errors.Is(err, claim_ad_refill_usecase.ErrNoAccount):
		return nil, connect.NewError(connect.CodeNotFound, claim_ad_refill_usecase.ErrNoAccount)
	case errors.Is(err, clicks.ErrBankFull):
		return nil, connect.NewError(connect.CodeFailedPrecondition, clicks.ErrBankFull)
	case errors.As(err, &used):
		return nil, refusal(used)
	default:
		return nil, err
	}
}

func refusal(used bonuses.AdRefillRefusal) *connect.Error {
	refused := connect.NewError(connect.CodeFailedPrecondition, bonuses.ErrAdRefillUsed)
	if detail, err := connect.NewErrorDetail(&planetv1.AdRefillRefusal{AvailableAtUnixMs: used.Until.UnixMilli()}); err == nil {
		refused.AddDetail(detail)
	}
	return refused
}
