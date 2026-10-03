package unsubscribed_handler

import (
	"context"
	"slices"

	"connectrpc.com/connect"

	marketingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/marketing/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
)

type UseCase interface {
	Execute(ctx context.Context, address subscriptions.Address) error
}

func New(useCase UseCase) UnsubscribedHandler {
	return UnsubscribedHandler{useCase: useCase}
}

type UnsubscribedHandler struct {
	useCase UseCase
}

// Brevo's payload says "unsubscribe"; the event it is configured with is "unsubscribed".
var unsubscribes = []string{"unsubscribe", "unsubscribed"}

func (h UnsubscribedHandler) Unsubscribed(
	ctx context.Context,
	req *connect.Request[marketingv1.UnsubscribedRequest],
) (*connect.Response[marketingv1.UnsubscribedResponse], error) {
	if !slices.Contains(unsubscribes, req.Msg.GetEvent()) {
		return connect.NewResponse(&marketingv1.UnsubscribedResponse{}), nil
	}

	address, err := subscriptions.AddressOf(req.Msg.GetEmail())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := h.useCase.Execute(ctx, address); err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it, and Brevo tries again.
	}
	return connect.NewResponse(&marketingv1.UnsubscribedResponse{}), nil
}
