package complete_discord_link_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	alertsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/alerts/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/alertsv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/usecases/complete_link_usecase"
)

var ErrDiscordRefused = errors.New("discord refused the link; start it again")

type UseCase interface {
	Execute(ctx context.Context, in complete_link_usecase.In) error
}

func New(useCase UseCase, serverURL string) CompleteDiscordLinkHandler {
	return CompleteDiscordLinkHandler{useCase: useCase, serverURL: serverURL}
}

type CompleteDiscordLinkHandler struct {
	useCase   UseCase
	serverURL string
}

func (h CompleteDiscordLinkHandler) CompleteDiscordLink(
	ctx context.Context,
	req *connect.Request[alertsv1.CompleteDiscordLinkRequest],
) (*connect.Response[alertsv1.CompleteDiscordLinkResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	err = h.useCase.Execute(ctx, complete_link_usecase.In{Account: account, Code: req.Msg.GetCode(), State: req.Msg.GetState()})
	switch {
	case errors.Is(err, links.ErrFlowInvalid):
		return nil, connect.NewError(connect.CodeFailedPrecondition, links.ErrFlowInvalid)
	case errors.Is(err, discord.ErrRefused):
		return nil, connect.NewError(connect.CodePermissionDenied, ErrDiscordRefused)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&alertsv1.CompleteDiscordLinkResponse{ServerUrl: h.serverURL}), nil
}
