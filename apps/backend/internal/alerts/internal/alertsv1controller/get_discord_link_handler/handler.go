package get_discord_link_handler

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	alertsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/alerts/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/alertsv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
)

type Query interface {
	Linked(ctx context.Context, account links.AccountID) (bool, error)
}

func New(query Query, serverURL string) GetDiscordLinkHandler {
	return GetDiscordLinkHandler{query: query, serverURL: serverURL}
}

type GetDiscordLinkHandler struct {
	query     Query
	serverURL string
}

func (h GetDiscordLinkHandler) GetDiscordLink(
	ctx context.Context,
	_ *connect.Request[alertsv1.GetDiscordLinkRequest],
) (*connect.Response[alertsv1.GetDiscordLinkResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}
	linked, err := h.query.Linked(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to read the link: %w", err)
	}
	return connect.NewResponse(&alertsv1.GetDiscordLinkResponse{Offered: true, Linked: linked, ServerUrl: h.serverURL}), nil
}
