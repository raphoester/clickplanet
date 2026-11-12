package get_recent_map_history_handler

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Query interface {
	RecentHistory(ctx context.Context) (*planetv1.MapHistory, time.Time, error)
}

func New(query Query, clock cptime.Clock) GetRecentMapHistoryHandler {
	return GetRecentMapHistoryHandler{query: query, clock: clock}
}

type GetRecentMapHistoryHandler struct {
	query Query
	clock cptime.Clock
}

func (h GetRecentMapHistoryHandler) GetRecentMapHistory(
	ctx context.Context,
	_ *connect.Request[planetv1.GetRecentMapHistoryRequest],
) (*connect.Response[planetv1.GetRecentMapHistoryResponse], error) {
	history, next, err := h.query.RecentHistory(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	res := connect.NewResponse(&planetv1.GetRecentMapHistoryResponse{History: history})
	age := max(int(next.Sub(h.clock.Now())/time.Second), 0)
	res.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, s-maxage=%d", age, age))
	return res, nil
}
