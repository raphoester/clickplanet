// Package announce_handler serves player.v1.PlayerService/Announce.
package announce_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/announce_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type UseCase interface {
	Execute(ctx context.Context, in announce_usecase.In) error
}

func New(useCase UseCase) AnnounceHandler {
	return AnnounceHandler{useCase: useCase}
}

type AnnounceHandler struct {
	useCase UseCase
}

// Announce tags the address the session interceptor read, which is the one Caddy put in X-Real-IP.
func (h AnnounceHandler) Announce(
	ctx context.Context,
	req *connect.Request[playerv1.AnnounceRequest],
) (*connect.Response[playerv1.AnnounceResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	err = h.useCase.Execute(ctx, announce_usecase.In{
		Account:   account,
		Country:   req.Msg.GetCountryId(),
		GuestName: req.Msg.GetGuestName(),
		IP:        cpctx.GetSourceIP(ctx),
	})
	if errors.Is(err, presence.ErrUnknownCountry) {
		return nil, connect.NewError(connect.CodeInvalidArgument, presence.ErrUnknownCountry)
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	return connect.NewResponse(&playerv1.AnnounceResponse{}), nil
}
