// Package get_profile_handler serves player.v1.PlayerService/GetProfile.
package get_profile_handler

import (
	"context"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
)

type UseCase interface {
	Execute(account players.AccountID) players.Profile
}

func New(useCase UseCase) GetProfileHandler {
	return GetProfileHandler{useCase: useCase}
}

type GetProfileHandler struct {
	useCase UseCase
}

func (h GetProfileHandler) GetProfile(
	ctx context.Context,
	_ *connect.Request[playerv1.GetProfileRequest],
) (*connect.Response[playerv1.GetProfileResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	return connect.NewResponse(&playerv1.GetProfileResponse{Profile: playermessage.Profile(h.useCase.Execute(account))}), nil
}
