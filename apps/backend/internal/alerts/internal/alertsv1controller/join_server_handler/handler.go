package join_server_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	alertsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/alerts/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/usecases/join_server_usecase"
)

var ErrNoToken = errors.New("no access token to join the server with")

type UseCase interface {
	Execute(ctx context.Context, in join_server_usecase.In) error
}

func New(useCase UseCase) JoinServerHandler {
	return JoinServerHandler{useCase: useCase}
}

type JoinServerHandler struct {
	useCase UseCase
}

func (h JoinServerHandler) JoinServer(
	ctx context.Context,
	req *connect.Request[alertsv1.JoinServerRequest],
) (*connect.Response[alertsv1.JoinServerResponse], error) {
	account, err := links.AccountIDOf(req.Msg.GetAccountId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	user, err := discord.UserIDOf(req.Msg.GetDiscordUserId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if req.Msg.GetAccessToken() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrNoToken)
	}

	err = h.useCase.Execute(ctx, join_server_usecase.In{
		Account: account, User: user, Token: discord.AccessToken(req.Msg.GetAccessToken()),
	})
	if errors.Is(err, discord.ErrRefused) {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&alertsv1.JoinServerResponse{}), nil
}
