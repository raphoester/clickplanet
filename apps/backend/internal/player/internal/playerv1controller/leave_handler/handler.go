// Package leave_handler serves player.v1.PlayerService/Leave.
package leave_handler

import (
	"context"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/caller"
)

type UseCase interface {
	Execute(ctx context.Context, account players.AccountID) error
}

func New(useCase UseCase) LeaveHandler {
	return LeaveHandler{useCase: useCase}
}

type LeaveHandler struct {
	useCase UseCase
}

func (h LeaveHandler) Leave(
	ctx context.Context,
	_ *connect.Request[playerv1.LeaveRequest],
) (*connect.Response[playerv1.LeaveResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	if err := h.useCase.Execute(ctx, account); err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	return connect.NewResponse(&playerv1.LeaveResponse{}), nil
}
