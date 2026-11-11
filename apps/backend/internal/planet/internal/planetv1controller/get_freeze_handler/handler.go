package get_freeze_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func New(freeze func() clicks.Freeze) GetFreezeHandler {
	return GetFreezeHandler{freeze: freeze}
}

// GetFreezeHandler answers whether the map takes no play now, by the freeze interceptor's own word: the rules in force
// freeze it, or an arena's freeze holds it.
type GetFreezeHandler struct {
	freeze func() clicks.Freeze
}

func (h GetFreezeHandler) GetFreeze(
	context.Context, *connect.Request[planetv1.GetFreezeRequest],
) (*connect.Response[planetv1.GetFreezeResponse], error) {
	return connect.NewResponse(&planetv1.GetFreezeResponse{Frozen: bool(h.freeze())}), nil
}
