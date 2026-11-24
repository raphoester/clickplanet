package stop_renewal_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	shopv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/shop/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/plans"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals/usecases/stop_renewal_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/shopv1controller/renewalmessage"
)

type UseCase interface {
	Execute(ctx context.Context, in stop_renewal_usecase.In) (renewals.Snapshot, error)
}

func New(useCase UseCase) StopRenewalHandler {
	return StopRenewalHandler{useCase: useCase}
}

type StopRenewalHandler struct {
	useCase UseCase
}

func (h StopRenewalHandler) StopRenewal(
	ctx context.Context,
	req *connect.Request[shopv1.StopRenewalRequest],
) (*connect.Response[shopv1.StopRenewalResponse], error) {
	stopped, err := h.useCase.Execute(ctx, stop_renewal_usecase.In{
		Account: cpctx.GetAccount(ctx), Linked: cpctx.GetLinked(ctx), Renewal: req.Msg.GetRenewalId(),
	})
	switch {
	// Another account's is none of the caller's: it reads as no renewal at all.
	case errors.Is(err, orders.ErrGuest), errors.Is(err, renewals.ErrUnknownRenewal), errors.Is(err, renewals.ErrNotYours):
		return nil, connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, plans.ErrOff), errors.Is(err, renewals.ErrEnded):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&shopv1.StopRenewalResponse{Renewal: renewalmessage.Renewal(stopped)}), nil
}
