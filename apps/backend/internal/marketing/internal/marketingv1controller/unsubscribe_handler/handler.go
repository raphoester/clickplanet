package unsubscribe_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	marketingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/marketing/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/marketingv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
)

type UseCase interface {
	Execute(ctx context.Context, account subscriptions.AccountID) error
}

func New(useCase UseCase) UnsubscribeHandler {
	return UnsubscribeHandler{useCase: useCase}
}

type UnsubscribeHandler struct {
	useCase UseCase
}

func (h UnsubscribeHandler) Unsubscribe(
	ctx context.Context,
	_ *connect.Request[marketingv1.UnsubscribeRequest],
) (*connect.Response[marketingv1.UnsubscribeResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	err = h.useCase.Execute(ctx, account)
	switch {
	case errors.Is(err, subscriptions.ErrAudienceUnreachable):
		return nil, connect.NewError(connect.CodeUnavailable, subscriptions.ErrAudienceUnreachable)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}

	return connect.NewResponse(&marketingv1.UnsubscribeResponse{}), nil
}
