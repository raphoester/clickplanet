package create_checkout_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	shopv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/shop/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders/usecases/create_checkout_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/plans"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/rpc_payables"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/shopv1controller/renewalmessage"
)

type UseCase interface {
	Execute(ctx context.Context, in create_checkout_usecase.In) (string, error)
}

func New(useCase UseCase) CreateCheckoutHandler {
	return CreateCheckoutHandler{useCase: useCase}
}

type CreateCheckoutHandler struct {
	useCase UseCase
}

func (h CreateCheckoutHandler) CreateCheckout(
	ctx context.Context,
	req *connect.Request[shopv1.CreateCheckoutRequest],
) (*connect.Response[shopv1.CreateCheckoutResponse], error) {
	target, err := orders.ParseTarget(req.Msg.GetSubmissionId(), req.Msg.GetFlag())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	page, err := h.useCase.Execute(ctx, create_checkout_usecase.In{
		Account: cpctx.GetAccount(ctx), Linked: cpctx.GetLinked(ctx), Plan: renewalmessage.PlanOf(req.Msg.GetPlan()),
		Target: target, StartsNow: req.Msg.GetStartsNow(),
	})
	switch {
	case errors.Is(err, orders.ErrGuest), errors.Is(err, rpc_payables.ErrNotYours):
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, rpc_payables.ErrUnknown):
		return nil, connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, plans.ErrOff), errors.Is(err, rpc_payables.ErrNotPayable), errors.Is(err, renewals.ErrRenewing):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, plans.ErrInvalidPlan), errors.Is(err, orders.ErrNoAgreement):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&shopv1.CreateCheckoutResponse{Url: page}), nil
}
