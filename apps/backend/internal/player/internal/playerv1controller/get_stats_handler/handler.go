// Package get_stats_handler serves player.v1.PlayerService/GetStats.
package get_stats_handler

import (
	"context"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
)

type UseCase interface {
	Execute(ctx context.Context, account players.AccountID) (players.Stats, error)
}

func New(useCase UseCase) GetStatsHandler {
	return GetStatsHandler{useCase: useCase}
}

type GetStatsHandler struct {
	useCase UseCase
}

func (h GetStatsHandler) GetStats(
	ctx context.Context,
	_ *connect.Request[playerv1.GetStatsRequest],
) (*connect.Response[playerv1.GetStatsResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	stats, err := h.useCase.Execute(ctx, account)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	return connect.NewResponse(&playerv1.GetStatsResponse{Stats: playermessage.Stats(stats)}), nil
}
