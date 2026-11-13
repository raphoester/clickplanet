package rpc_planet_freeze

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

type Planet interface {
	GetFreeze(context.Context, *connect.Request[planetv1.GetFreezeRequest]) (*connect.Response[planetv1.GetFreezeResponse], error)
}

func New(planet Planet) Freeze {
	return Freeze{planet: planet}
}

// Freeze asks planet whether its map takes no play now, once per claim, opening and answer: they are few, and an
// arena's freeze lasts seconds.
type Freeze struct {
	planet Planet
}

func (f Freeze) Frozen(ctx context.Context) (bool, error) {
	res, err := f.planet.GetFreeze(ctx, connect.NewRequest(&planetv1.GetFreezeRequest{}))
	if err != nil {
		return false, fmt.Errorf("failed to ask planet whether its map takes play: %w", err)
	}
	return res.Msg.GetFrozen(), nil
}
