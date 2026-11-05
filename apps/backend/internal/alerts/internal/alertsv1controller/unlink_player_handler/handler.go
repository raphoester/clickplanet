package unlink_player_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	alertsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/alerts/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
)

var ErrDiscordRefused = errors.New("discord would not take the flag's role off the user; the link is kept")

type UseCase interface {
	Execute(ctx context.Context, account links.AccountID) (bool, error)
}

func New(useCase UseCase) UnlinkPlayerHandler {
	return UnlinkPlayerHandler{useCase: useCase}
}

type UnlinkPlayerHandler struct {
	useCase UseCase
}

func (h UnlinkPlayerHandler) UnlinkPlayer(
	ctx context.Context,
	req *connect.Request[alertsv1.UnlinkPlayerRequest],
) (*connect.Response[alertsv1.UnlinkPlayerResponse], error) {
	account, err := links.AccountIDOf(req.Msg.GetAccountId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	unlinked, err := h.useCase.Execute(ctx, account)
	if errors.Is(err, discord.ErrRefused) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, ErrDiscordRefused)
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&alertsv1.UnlinkPlayerResponse{Unlinked: unlinked}), nil
}
