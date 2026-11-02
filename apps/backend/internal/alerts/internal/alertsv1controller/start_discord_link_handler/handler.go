package start_discord_link_handler

import (
	"context"

	"connectrpc.com/connect"

	alertsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/alerts/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/alertsv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
)

type UseCase interface {
	Execute(ctx context.Context, account links.AccountID) (string, error)
}

func New(useCase UseCase) StartDiscordLinkHandler {
	return StartDiscordLinkHandler{useCase: useCase}
}

type StartDiscordLinkHandler struct {
	useCase UseCase
}

func (h StartDiscordLinkHandler) StartDiscordLink(
	ctx context.Context,
	_ *connect.Request[alertsv1.StartDiscordLinkRequest],
) (*connect.Response[alertsv1.StartDiscordLinkResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}
	url, err := h.useCase.Execute(ctx, account)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&alertsv1.StartDiscordLinkResponse{AuthorizationUrl: url}), nil
}
