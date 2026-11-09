package rpc_planet_arenas

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
)

type Planet interface {
	GetRuleSet(ctx context.Context, req *connect.Request[planetv1.GetRuleSetRequest],
	) (*connect.Response[planetv1.GetRuleSetResponse], error)
}

const askTimeout = 2 * time.Second

func New(planet Planet) *Arenas {
	return &Arenas{planet: planet}
}

// Arenas reads whether a set makes an arena battle, from planet's internal service.
type Arenas struct {
	planet Planet
}

// Arena is false for a set planet keeps none of: planet refuses it and plays on, on the map as it is.
func (s *Arenas) Arena(ctx context.Context, set schedule.SetName) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := s.planet.GetRuleSet(ctx, connect.NewRequest(&planetv1.GetRuleSetRequest{Name: string(set)}))
	switch {
	case connect.CodeOf(err) == connect.CodeNotFound:
		return false, nil
	case err != nil:
		return false, fmt.Errorf("failed to ask planet whether %q is fought in an arena: %w", set, err)
	}
	return res.Msg.GetPlayed().GetArena().GetOn(), nil
}
