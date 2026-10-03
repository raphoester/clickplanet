package subscribe_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	marketingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/marketing/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/marketingv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/marketingv1controller/marketingmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
)

type UseCase interface {
	Execute(ctx context.Context, account subscriptions.AccountID, address string) (subscriptions.State, error)
}

func New(useCase UseCase) SubscribeHandler {
	return SubscribeHandler{useCase: useCase}
}

type SubscribeHandler struct {
	useCase UseCase
}

func (h SubscribeHandler) Subscribe(
	ctx context.Context,
	req *connect.Request[marketingv1.SubscribeRequest],
) (*connect.Response[marketingv1.SubscribeResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	state, err := h.useCase.Execute(ctx, account, req.Msg.GetAddress())
	switch {
	case errors.Is(err, subscriptions.ErrAddressInvalid):
		return nil, connect.NewError(connect.CodeInvalidArgument, subscriptions.ErrAddressInvalid)
	case errors.Is(err, subscriptions.ErrNotLinked):
		return nil, connect.NewError(connect.CodePermissionDenied, subscriptions.ErrNotLinked)
	case errors.Is(err, subscriptions.ErrAlreadySubscribed):
		return nil, connect.NewError(connect.CodeFailedPrecondition, subscriptions.ErrAlreadySubscribed)
	case errors.Is(err, subscriptions.ErrAudienceUnreachable):
		return nil, connect.NewError(connect.CodeUnavailable, subscriptions.ErrAudienceUnreachable)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	return connect.NewResponse(&marketingv1.SubscribeResponse{State: marketingmessage.StateOf(state)}), nil
}
