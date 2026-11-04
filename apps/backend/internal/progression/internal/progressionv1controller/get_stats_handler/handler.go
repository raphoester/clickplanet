package get_stats_handler

import (
	"context"

	"connectrpc.com/connect"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/caller"
)

type Query interface {
	Stats(ctx context.Context, account accounts.AccountID) (*progressionv1.GetStatsResponse, error)
}

func New(query Query) GetStatsHandler {
	return GetStatsHandler{query: query}
}

type GetStatsHandler struct {
	query Query
}

func (h GetStatsHandler) GetStats(
	ctx context.Context,
	_ *connect.Request[progressionv1.GetStatsRequest],
) (*connect.Response[progressionv1.GetStatsResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	stats, err := h.query.Stats(ctx, account)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(stats), nil
}
