package get_discord_grant_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/grants"
)

type UseCase interface {
	Execute(ctx context.Context, id grants.GrantID) (grants.Grant, error)
}

func New(useCase UseCase) GetDiscordGrantHandler {
	return GetDiscordGrantHandler{useCase: useCase}
}

// Served on the internal listener alone: its answer carries a token that may add a user to the game's server.
type GetDiscordGrantHandler struct {
	useCase UseCase
}

func (h GetDiscordGrantHandler) GetDiscordGrant(
	ctx context.Context,
	req *connect.Request[authv1.GetDiscordGrantRequest],
) (*connect.Response[authv1.GetDiscordGrantResponse], error) {
	id, err := grants.GrantIDOf(req.Msg.GetGrantId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	grant, err := h.useCase.Execute(ctx, id)
	switch {
	case errors.Is(err, grants.ErrGrantNotFound), errors.Is(err, grants.ErrGrantExpired):
		return nil, connect.NewError(connect.CodeNotFound, err)
	case err != nil:
		return nil, fmt.Errorf("failed to read the grant: %w", err)
	}

	kept := grant.TakeSnapshot()
	return connect.NewResponse(&authv1.GetDiscordGrantResponse{
		AccountId: kept.Account.String(), DiscordUserId: kept.User, AccessToken: kept.Token,
	}), nil
}
