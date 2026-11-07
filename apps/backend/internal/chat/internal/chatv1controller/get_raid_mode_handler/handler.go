package get_raid_mode_handler

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
)

type Query interface {
	RaidMode(ctx context.Context) (*chatv1.GetRaidModeResponse, error)
}

func New(query Query) GetRaidModeHandler {
	return GetRaidModeHandler{query: query}
}

type GetRaidModeHandler struct {
	query Query
}

func (h GetRaidModeHandler) GetRaidMode(
	ctx context.Context,
	_ *connect.Request[chatv1.GetRaidModeRequest],
) (*connect.Response[chatv1.GetRaidModeResponse], error) {
	mode, err := h.query.RaidMode(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read the raid mode: %w", err)
	}

	res := connect.NewResponse(mode)
	res.Header().Set("Cache-Control", "no-store")
	return res, nil
}
