package get_player_link_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	alertsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/alerts/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/alertsv1controller/get_player_link_handler/player_link_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
)

type Query interface {
	LinkOf(ctx context.Context, account links.AccountID) (*alertsv1.DiscordLink, error)
}

func New(query Query) GetPlayerLinkHandler {
	return GetPlayerLinkHandler{query: query}
}

type GetPlayerLinkHandler struct {
	query Query
}

func (h GetPlayerLinkHandler) GetPlayerLink(
	ctx context.Context,
	req *connect.Request[alertsv1.GetPlayerLinkRequest],
) (*connect.Response[alertsv1.GetPlayerLinkResponse], error) {
	account, err := links.AccountIDOf(req.Msg.GetAccountId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	link, err := h.query.LinkOf(ctx, account)
	if errors.Is(err, player_link_query.ErrNoLink) {
		return connect.NewResponse(&alertsv1.GetPlayerLinkResponse{}), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the link: %w", err)
	}
	return connect.NewResponse(&alertsv1.GetPlayerLinkResponse{Link: link}), nil
}
