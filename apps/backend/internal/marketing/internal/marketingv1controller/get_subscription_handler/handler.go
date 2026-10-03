package get_subscription_handler

import (
	"context"

	"connectrpc.com/connect"

	marketingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/marketing/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/marketingv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/marketingv1controller/marketingmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
)

type UseCase interface {
	Execute(ctx context.Context, account subscriptions.AccountID) (subscriptions.Status, error)
}

func New(useCase UseCase) GetSubscriptionHandler {
	return GetSubscriptionHandler{useCase: useCase}
}

type GetSubscriptionHandler struct {
	useCase UseCase
}

func (h GetSubscriptionHandler) GetSubscription(
	ctx context.Context,
	_ *connect.Request[marketingv1.GetSubscriptionRequest],
) (*connect.Response[marketingv1.GetSubscriptionResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	status, err := h.useCase.Execute(ctx, account)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}

	res := connect.NewResponse(&marketingv1.GetSubscriptionResponse{
		State: marketingmessage.StateOf(status.State), Address: string(status.Address),
	})
	res.Header().Set("Cache-Control", "no-store")
	return res, nil
}
