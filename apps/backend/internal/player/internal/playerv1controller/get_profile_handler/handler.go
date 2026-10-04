package get_profile_handler

import (
	"context"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/caller"
)

type Query interface {
	Profile(ctx context.Context, account players.AccountID) (*playerv1.GetProfileResponse, error)
}

func New(query Query) GetProfileHandler {
	return GetProfileHandler{query: query}
}

type GetProfileHandler struct {
	query Query
}

func (h GetProfileHandler) GetProfile(
	ctx context.Context,
	_ *connect.Request[playerv1.GetProfileRequest],
) (*connect.Response[playerv1.GetProfileResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	profile, err := h.query.Profile(ctx, account)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(profile), nil
}
